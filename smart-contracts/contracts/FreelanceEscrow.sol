// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract FreelanceEscrow {
    address public aiOracle; // The backend server address

    struct Escrow {
        address client;
        address freelancer;
        uint256 totalAmount;
        uint256 amountReleased;
        bool isActive;
    }

    // Maps escrow ID (string) to Escrow struct
    mapping(string => Escrow) public escrows;

    // Maps address to reputation score (0 to 50, meaning 0.0 to 5.0)
    mapping(address => uint256) public reputation;
    mapping(address => bool) public hasReputation;

    event EscrowCreated(string escrowId, address client, address freelancer, uint256 amount);
    event FundsReleased(string escrowId, uint256 amount, address to);
    event DisputeResolved(string escrowId, uint256 freelancerAmount, uint256 clientAmount);
    event ReputationUpdated(address user, uint256 newScore);

    modifier onlyOracle() {
        // require(msg.sender == aiOracle, "Only AI Oracle can call this"); // Commented out for Hackathon Demo UI ease
        _;
    }

    constructor() {
        aiOracle = msg.sender; // The deployer is the AI Oracle
    }

    function initReputation(address user) internal {
        if (!hasReputation[user]) {
            reputation[user] = 50; // Start at 5.0
            hasReputation[user] = true;
        }
    }

    function createEscrow(string memory escrowId, address freelancer) external payable {
        require(msg.value > 0, "Must deposit funds");
        require(escrows[escrowId].client == address(0), "Escrow already exists");

        initReputation(msg.sender);
        initReputation(freelancer);

        escrows[escrowId] = Escrow({
            client: msg.sender,
            freelancer: freelancer,
            totalAmount: msg.value,
            amountReleased: 0,
            isActive: true
        });

        emit EscrowCreated(escrowId, msg.sender, freelancer, msg.value);
    }

    function approveMilestone(string memory escrowId, uint256 amount) external onlyOracle {
        Escrow storage e = escrows[escrowId];
        require(e.isActive, "Escrow is not active");
        require(e.amountReleased + amount <= e.totalAmount, "Amount exceeds total");

        e.amountReleased += amount;
        payable(e.freelancer).transfer(amount);

        // Increase freelancer reputation slightly
        if (reputation[e.freelancer] < 50) {
            reputation[e.freelancer] += 1;
            emit ReputationUpdated(e.freelancer, reputation[e.freelancer]);
        }

        if (e.amountReleased == e.totalAmount) {
            e.isActive = false;
        }

        emit FundsReleased(escrowId, amount, e.freelancer);
    }

    function resolveDispute(string memory escrowId, uint256 freelancerAmount, uint256 clientAmount) external onlyOracle {
        Escrow storage e = escrows[escrowId];
        require(e.isActive, "Escrow is not active");
        require(e.amountReleased + freelancerAmount + clientAmount <= e.totalAmount, "Amounts exceed balance");

        e.isActive = false; // Close the escrow

        if (freelancerAmount > 0) {
            payable(e.freelancer).transfer(freelancerAmount);
        }
        if (clientAmount > 0) {
            payable(e.client).transfer(clientAmount);
        }

        // Adjust reputation based on payout split
        if (freelancerAmount > clientAmount) {
            // Freelancer won
            if (reputation[e.client] >= 2) reputation[e.client] -= 2;
        } else {
            // Client won
            if (reputation[e.freelancer] >= 2) reputation[e.freelancer] -= 2;
        }

        emit ReputationUpdated(e.client, reputation[e.client]);
        emit ReputationUpdated(e.freelancer, reputation[e.freelancer]);

        emit DisputeResolved(escrowId, freelancerAmount, clientAmount);
    }

    // Getter for frontend
    function getReputation(address user) external view returns (uint256) {
        if (!hasReputation[user]) return 50;
        return reputation[user];
    }

    // Allow contract to receive ETH
    receive() external payable {}
}
