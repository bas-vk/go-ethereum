package core

/*
func TestCreation(t *testing.T) {
	var (
		db, _               = ethdb.NewMemDatabase()
		key1, _             = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1               = crypto.PubkeyToAddress(key1.PublicKey)
		addr1Nonce   uint64 = 0
		key2, _             = crypto.HexToECDSA("c71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr2               = crypto.PubkeyToAddress(key2.PublicKey)
		addr2Nonce   uint64 = 0
		key3, _             = crypto.HexToECDSA("d71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr3               = crypto.PubkeyToAddress(key3.PublicKey)
		addr3Nonce   uint64 = 0
		chainConfig         = &core.ChainConfig{HomesteadBlock: new(big.Int)}
		makerAddress        = common.Address{20}
	)
	defer db.Close()

	core.WriteGenesisBlockForTesting(db,
		core.GenesisAccount{addr1, big.NewInt(1000000), nil, nil},
		core.GenesisAccount{addr2, big.NewInt(1000000), nil, nil},
		core.GenesisAccount{addr3, big.NewInt(1000000), nil, nil},
		core.GenesisAccount{makerAddress, new(big.Int), common.FromHex(DeployCode), map[string]string{
			"0000000000000000000000000000000000000000000000000000000000000001": "01",
			// add addr1 as a voter
			common.Bytes2Hex(crypto.Keccak256(append(common.LeftPadBytes(addr1[:], 32), common.LeftPadBytes([]byte{2}, 32)...))): "01",
		}},
	)

	evmux := &event.TypeMux{}
	blockchain, err := core.NewBlockChain(db, chainConfig, core.FakePow{}, evmux)
	if err != nil {
		t.Fatal(err)
	}

	maker := NewBlockMaker(chainConfig, makerAddress, blockchain, db, &event.TypeMux{})

	myChainHead := findDecendant(maker.CanonHash(), blockchain).Hash()

	// allow addr2 and 3 to vote
	tx0, _ := maker.AddVoter(addr2, addr1Nonce, key1)
	addr1Nonce++

	tx1, _ := maker.AddVoter(addr3, addr1Nonce, key1)
	addr1Nonce++

	// vote for my own header to be the head of the canonical chain
	tx2, _ := maker.Vote(myChainHead, addr1Nonce, key1)
	addr1Nonce++

	block, _ := maker.Create(types.Transactions{tx0, tx1, tx2})
	if i, err := blockchain.InsertChain(types.Blocks{block}); err != nil {
		t.Fatalf("insert error (block %d): %v\n", i, err)
		return
	}

	// verify that myChainHead has become the new canonical chain
	if maker.CanonHash() != myChainHead {
		t.Fatalf("expected %x to be the canonical hash, got %x", myChainHead, maker.CanonHash())
	}

	// let 2 blocks compete
	//                        blockA
	//                       /      \
	// genesis -- myChainHead        -- block
	//                       \      /
	//                        blockB
	myChainHead = findDecendant(maker.CanonHash(), blockchain).Hash()

	vote0, _ := maker.Vote(myChainHead, addr2Nonce, key2)
	addr2Nonce++

	// when blocks are created within the same second the blockmaker adds 1 second the the parent block timestamp
	// and uses that as the timestamp for the new block. This causes the blockchain to queue it since the timestamp
	// is within the future. This normally doesn't happen (ofter) but in this test is does. Therefore a small sleep is
	// introduced.
	time.Sleep(1100 * time.Millisecond)

	blockA, _ := maker.Create(types.Transactions{vote0})
	chainA := types.Blocks{blockA}
	if i, err := blockchain.InsertChain(chainA); err != nil {
		t.Fatalf("insert error (block %d): %v\n", i, err)
		return
	}

	vote1, _ := maker.Vote(maker.CanonHash(), addr3Nonce, key3)
	addr3Nonce++
	blockB, _ := maker.Create(types.Transactions{vote1})
	chainB := types.Blocks{blockB}
	if i, err := blockchain.InsertChain(chainB); err != nil {
		t.Fatalf("insert error (block %d): %v\n", i, err)
		return
	}

	t.Logf("voting on block (%x) twice\n", blockA.Hash())
	t.Logf("voting on block (%x) once\n", blockB.Hash())
	vote0, _ = maker.Vote(blockA.Hash(), addr1Nonce, key1)
	vote1, _ = maker.Vote(blockA.Hash(), addr2Nonce, key2)
	vote2, _ := maker.Vote(blockB.Hash(), addr3Nonce, key3)
	addr1Nonce++
	addr2Nonce++
	addr3Nonce++

	winnerHash := chainA[0].Hash()

	block, _ = maker.Create(types.Transactions{vote0, vote1, vote2})
	if i, err := blockchain.InsertChain(types.Blocks{block}); err != nil {
		t.Fatalf("insert error (block %d): %v\n", i, err)
		return
	}

	blck := blockchain.CurrentBlock()
	for blck != nil {
		blck = blockchain.GetBlockByHash(blck.ParentHash())
	}

	// verify that chain1 "won"
	block, _ = maker.Create(nil)
	if winnerHash != block.ParentHash() {
		t.Errorf("expected %x to be canonical, got %x", winnerHash, block.ParentHash())
	}
}
*/
