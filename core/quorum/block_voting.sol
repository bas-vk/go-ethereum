// Implements a block voting algorithm to reach consensus.
//
// To vote for a block the sender must be allowed to vote. When deployed the
// deployer is the only party that is allowed to vote and can add new voters.
// Note that voters can add new voters and thus have the abbility to add multiple
// voter accounts that they control. This gives them the possibility to vote
// multiple times for a particular block. Therefore voters must be trusted.
contract BlockVoting {
    // Raised when a sender that is not allowed to vote makes a vote
    event InvalidSenderVote(address sender, uint blockNumber);

    // Raised when a vote is made
    event Vote(address indexed sender, uint blockNumber, bytes32 blockHash);

    // The period in which voters can vote for a block that is selected
    // as the new head of the chain.
	struct Period {
	    // number of times a block is voted for
		mapping(bytes32 => uint) entries;

		// blocks up for voting
		bytes32[] indices;
	}

    // Collection of vote rounds.
	Period[] periods;

    // canonical hash must have as least voteThreshold votes before its considered valid
	uint public voteThreshold;

	// number of voters
	uint public voterCount;

    // Collection of addresses that are allowed to vote.
    mapping(address => bool) public canVote;

    // Only allow addresses that currently allowed to vote.
	modifier mustBeVoter() {
		if( canVote[msg.sender] ) {
		    _
		} else {
		    InvalidSenderVote(msg.sender, block.number-1);
		}
	}

	function BlockVoting() {
		canVote[msg.sender] = true;
		voterCount = 1;
	}

    // Set a new vote threshold. The canonical hash must have at least the given
    // threshold number of votes before it's considered valid.
	function setVoteThreshold(uint threshold) mustBeVoter {
	    voteThreshold = threshold;
	}

    // Make a vote to select a particular block as head for the previous head.
    // Only senders that are added through the addVoter are allowed to make a vote.
    // TODO: discuss if we only allow 1 vote per voter (this can deadlock the system)
	function vote(bytes32 hash) mustBeVoter {
	    // start new period if this is the first transaction in the new block.
		if (periods.length < block.number) {
		    periods.length++;
		}

		// select the previous voting round.
		Period period = periods[block.number-1];

		// new block hash entry
		if(period.entries[hash] == 0) period.indices.push(hash);

		// vote
		period.entries[hash]++;

		// log vote
		Vote(msg.sender, block.number, hash);
	}

    // Get the "winning" block hash of the previous voting round.
    // Ensure the "winning" block has at least voteThreshold votes.
	function getCanonHash() constant returns(bytes32) {
		Period period = periods[periods.length-1];

		bytes32 best;
		for(uint i = 0; i < period.indices.length; i++) {
			if(period.entries[best] < period.entries[period.indices[i]]
			&& period.entries[period.indices[i]] >= voteThreshold) {
				best = period.indices[i];
			}
		}
		return best;
	}

	// Add an party that is allowed to make a vote.
	// Only current voters are allowed to add a new voter.
	function addVoter(address addr) mustBeVoter {
		canVote[addr] = true;
		voterCount++;
	}

	// Remove a party that is allowed to vote.
	// Note, a voter can remove it self as a voter!
	function removeVoter(address addr) mustBeVoter {
	    // don't let the last voter remove it self which can cause the
	    // algorithm to stall.
	    if (voterCount == 1) throw;

	    delete canVote[addr];
	    voterCount--;
	}

    // Number of voting rounds.
	function getSize() constant returns(uint) {
		return periods.length;
	}

    // Return a blockhash by period and index.
	function getEntry(uint p, uint n) constant returns(bytes32) {
		Period period = periods[p];
		return period.indices[n];
	}
}