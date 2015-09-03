// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
package useragent

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	qml "gopkg.in/qml.v1"
)

const (
	MaxPasswordAttempts = 3 // number of times the password is request by the user before giving up unlocking the acc

	txConfirmationQmlResource = `
import QtQuick 2.3
import QtQuick.Controls 1.4
import QtQuick.Controls.Styles 1.4
import QtQuick.Layouts 1.0
import QtQuick.Window 2.1

ApplicationWindow {
    id: applicationWindow
    title: mainWindowTitle
    visible: true

    property int margin: 8
    width: mainLayout.implicitWidth + 2 * margin
    height: mainLayout.implicitHeight + 2 * margin

    // no resizing
    minimumWidth: mainLayout.minimumWidth + 2 * margin
    maximumWidth: mainLayout.minimumWidth + 2 * margin
    minimumHeight: mainLayout.Layout.minimumHeight + 2 * margin
    maximumHeight: mainLayout.Layout.minimumHeight + 2 * margin

    // user actions
    signal accept(string password)
    signal cancel

    onClosing: {
        cancel()
    }

    x: (Screen.width - width) / 2
    y: (Screen.Height - height) / 2

    ColumnLayout {
        id: mainLayout
        anchors.fill: parent
        anchors.margins: margin


        property int minimumWidth: 550

        GroupBox {
            id: messageSection
            visible: !requestPassword
            title: "Geth needs your approval."

            Text {
                width: mainLayout.minimumWidth - 2 * margin
                text: message
                wrapMode: Text.WordWrap
            }
        }

        GroupBox {
            id: passwordSection
            visible: requestPassword
            title: "Geth needs to unlock your account " + passwordAttempt + "."

            RowLayout {
                width: mainLayout.minimumWidth - 2 * margin
                Label {
                    id: passwordLabel
                    text: "Password"
                }
                TextField {
                    id: passwordTextField
                    echoMode: TextInput.Password
                    Layout.fillWidth: true
                    placeholderText: account
                }
            }
        }

        GroupBox {
            id: buttonSection
            anchors.right: parent.right
            RowLayout {
                Button {
                    id: rejectButton
                    text: qsTr("Reject")
                    onClicked: {
                        cancel()
                    }
                    style: ButtonStyle {
                        label: Text {
                            renderType: Text.NativeRendering
                            verticalAlignment: Text.AlignVCenter
                            horizontalAlignment: Text.AlignHCenter
                            color: "#fff"
                            text: control.text
                        }
                        background: Rectangle {
                            implicitWidth: 100
                            implicitHeight: 25
                            border.width: control.activeFocus ? 2 : 1
                            border.color: "#888"
                            radius: 4
                            color: "red"
                            gradient: Gradient {
                                GradientStop { position: 0 ; color: control.pressed ? "#e40a0a" : "#f62b2b" }
                                GradientStop { position: 1 ; color: control.pressed ? "#9f0202" : "#d20202" }
                            }
                        }
                    }
                }

                Button {
                    id: approveButton
                    text: qsTr("Approve")
                    onClicked: {
                        accept(passwordTextField.text)
                    }
                    style: ButtonStyle {
                        label: Text {
                            renderType: Text.NativeRendering
                            verticalAlignment: Text.AlignVCenter
                            horizontalAlignment: Text.AlignHCenter
                            color: "#fff"
                            text: control.text
                        }
                        background: Rectangle {
                            implicitWidth: 100
                            implicitHeight: 25
                            border.width: control.activeFocus ? 2 : 1
                            border.color: "#888"
                            radius: 4
                            color: "green"
                            gradient: Gradient {
                                GradientStop { position: 0 ; color: control.pressed ? "#36780f" : "#4ba614" }
                                GradientStop { position: 1 ; color: control.pressed ? "#005900" : "#008c00" }
                            }
                        }
                    }
                }
            }
        }
    }
}
`
)

var requestQueue chan interface{}

type passwordResponse struct {
	Cancelled bool
	Password  string
}

type passwordRequest struct {
	Account string
	Attempt int
	Result  chan passwordResponse
}

type confirmTransactionRequest struct {
	Tx            string
	From          string
	RequestPasswd bool
	Attempt       int
	Result        chan confirmTransactionResponse
}

type confirmTransactionResponse struct {
	Approved bool
	Password string
}

// StartQMLFrontend initializes the QML runtime, starts the QML loop and handles frontend requests. It must be called
// once from the main go routine and before any QML frontend instances are used. Note, this will block!
func StartQMLFrontend() {
	requestQueue = make(chan interface{})

	qml.Run(func() error {
		qmlEngine := qml.NewEngine()
		txConfirmationQmlObject, err := qmlEngine.LoadString("txConfirmationQmlResource", txConfirmationQmlResource)
		if err != nil {
			return err
		}

		for {
			select {
			case r := <-requestQueue:
				switch req := r.(type) {
				case passwordRequest:
					ctx := qmlEngine.Context()
					ctx.SetVar("mainWindowTitle", "Unlock account")
					ctx.SetVar("message", "password unlock")
					ctx.SetVar("account", req.Account)
					ctx.SetVar("requestPassword", true)
					ctx.SetVar("passwordAttempt", fmt.Sprintf("(attempt %d/%d)", req.Attempt, MaxPasswordAttempts))
					win := txConfirmationQmlObject.CreateWindow(ctx)

					var cancelled bool
					var passwd string
					win.On("accept", func(p string) {
						passwd = p
						win.Destroy()
					})

					win.On("cancel", func() {
						cancelled = true
						win.Destroy()
					})

					win.Show()
					win.Wait()

					req.Result <- passwordResponse{cancelled, passwd}
				case confirmTransactionRequest:
					ctx := qmlEngine.Context()
					ctx.SetVar("mainWindowTitle", "Transaction confirmation")
					ctx.SetVar("message", req.Tx)
					ctx.SetVar("account", req.From)
					ctx.SetVar("requestPassword", req.RequestPasswd)
					ctx.SetVar("passwordAttempt", fmt.Sprintf("(attempt %d/%d)", req.Attempt, MaxPasswordAttempts))
					win := txConfirmationQmlObject.CreateWindow(ctx)

					var confirmed bool
					var passwd string
					win.On("accept", func(p string) {
						confirmed = true
						passwd = p
						win.Destroy()
					})

					win.On("cancel", func() {
						confirmed = false
						win.Destroy()
					})

					win.Show()
					win.Wait()

					req.Result <- confirmTransactionResponse{confirmed, passwd}
				}
			}
		}

		return nil
	})
}

type qmlFrontend struct {
	mgr *accounts.Manager
}

func NewQMLFrontend(mgr *accounts.Manager) *qmlFrontend {
	return &qmlFrontend{mgr}
}

func (fe *qmlFrontend) UnlockAccount(address []byte) bool {
	acc := common.BytesToAddress(address)
	c := make(chan passwordResponse)
	defer close(c)

	r := passwordRequest{
		Account: acc.Hex(),
		Result:  c,
	}

	for i := 1; i <= MaxPasswordAttempts; i++ {
		r.Attempt = i

		requestQueue <- r
		res := <-c

		if res.Cancelled {
			return false
		}

		if err := fe.mgr.Unlock(acc, res.Password); err == nil {
			return true
		}
	}

	return false
}

func (fe *qmlFrontend) ConfirmTransaction(tx string) (confirmed bool) {
	c := make(chan confirmTransactionResponse)
	defer close(c)

	r := confirmTransactionRequest{
		Tx:            tx,
		Result:        c,
		RequestPasswd: false,
	}

	for i := 1; i <= MaxPasswordAttempts; i++ {
		r.Attempt = i

		requestQueue <- r
		response := <-c

		return response.Approved
	}

	return false
}
