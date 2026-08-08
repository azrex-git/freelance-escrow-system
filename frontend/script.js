let escrowAmount = 0;
let reputation = 4.5;
let isEscrowActive = false;
let currentEscrowId = "";
let currentMilestoneId = "";
let userWallet = "";
let escrowContract = null;
const contractAddress = "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512";
const contractABI = [
  "function createEscrow(string escrowId, address freelancer) external payable",
  "function approveMilestone(string escrowId, uint256 amount) external",
  "function getReputation(address user) external view returns (uint256)"
async function initWeb3() {
  try {
    // Connect silently to the local Hardhat node
    const provider = new ethers.JsonRpcProvider("http://127.0.0.1:8545");
    // Hardhat Account #0 private key
    const privateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"; 
    const signer = new ethers.Wallet(privateKey, provider);
    
    userWallet = signer.address;
    escrowContract = new ethers.Contract(contractAddress, contractABI, signer);
    
    // Fetch reputation
    const repScore = await escrowContract.getReputation(userWallet);
    document.getElementById('rep').innerText = (Number(repScore) / 10).toFixed(1);
  } catch (e) {
    console.error("Failed to connect to local Web3 node. Ensure Hardhat is running.", e);
  }
}

// Auto-initialize on load
window.addEventListener('load', initWeb3);

// Helpers
const showMsg = (id, msg, type = 'success') => {
  const el = document.getElementById(id);
  el.innerText = msg;
  el.className = `status-msg show text-${type}`;
};

const setLoading = (btnId, isLoading) => {
  const btn = document.getElementById(btnId);
  const text = btn.querySelector('.btn-text');
  const loader = btn.querySelector('.loader');
  
  if (isLoading) {
    btn.disabled = true;
    btn.style.opacity = '0.8';
    text.classList.add('hidden');
    loader.classList.remove('hidden');
  } else {
    btn.disabled = false;
    btn.style.opacity = '1';
    text.classList.remove('hidden');
    loader.classList.add('hidden');
  }
};

async function deposit() {
  const amountInput = document.getElementById("amount");
  const amount = parseFloat(amountInput.value);
  const clientRules = document.getElementById("clientRules").value;
  
  if (!userWallet || !escrowContract) {
    showMsg("escrowStatus", "Please wait for blockchain to initialize...", "warning");
    return;
  }
  
  if (isNaN(amount) || amount <= 0) {
    showMsg("escrowStatus", "Please enter a valid amount.", "error");
    return;
  }

  setLoading('btn-deposit', true);

  try {
    // 1. Create Web2 Escrow tracking on Go Backend
    const response = await fetch('/api/escrow', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        client_id: userWallet,
        freelancer_id: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // Hardhat Account #1 as freelancer mock
        milestones: [{ 
          description: "Main Deliverable", 
          amount: amount,
          client_instructions: clientRules
        }]
      })
    });
    
    if (response.ok) {
      const data = await response.json();
      currentEscrowId = data.id;
      currentMilestoneId = data.milestones[0].id;
      
      // 2. Lock funds in Web3 Smart Contract
      const tx = await escrowContract.createEscrow(currentEscrowId, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", {
        value: ethers.parseEther(amount.toString())
      });
      showMsg("escrowStatus", "Waiting for blockchain confirmation...", "success");
      await tx.wait();

      escrowAmount = amount;
      isEscrowActive = true;
      showMsg("escrowStatus", `Escrow Locked! TX: ${tx.hash.substring(0,10)}...`, "success");
      amountInput.value = '';
    } else {
      showMsg("escrowStatus", "Failed to create Web2 Escrow tracking.", "error");
    }
  } catch (error) {
    console.error(error);
    showMsg("escrowStatus", "Transaction failed or rejected.", "error");
  } finally {
    setLoading('btn-deposit', false);
  }
}

async function checkAI() {
  const work = document.getElementById("work").value;
  const fileInput = document.getElementById("fileUpload");
  
  if (!isEscrowActive) {
    showMsg("aiResult", "No active escrow. Client must deposit first.", "warning");
    return;
  }
  
  if (!work.trim() && (!fileInput.files || fileInput.files.length === 0)) {
    showMsg("aiResult", "Please provide work details or upload a file.", "error");
    return;
  }

  setLoading('btn-ai', true);
  
  let fileName = "";
  let fileContent = "";
  
  if (fileInput.files && fileInput.files.length > 0) {
    const file = fileInput.files[0];
    fileName = file.name;
    try {
      fileContent = await file.text();
    } catch (e) {
      showMsg("aiResult", "Error reading file. Only text/code files are supported in this demo.", "error");
      setLoading('btn-ai', false);
      return;
    }
  }

  try {
    const response = await fetch(`/api/escrow/${currentEscrowId}/milestone/${currentMilestoneId}/evaluate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        submitted_work: work,
        file_name: fileName,
        file_content: fileContent
      })
    });
    
    const data = await response.json().catch(() => ({}));

    if (response.ok) {
      const match = data.match_percentage || 0;
      const feedback = data.feedback || "Evaluation complete.";
      
      if (match < 50) {
        showMsg("aiResult", `🤖 AI Judge: ${match}% Match. ${feedback}`, "error");
      } else if (match < 85) {
        showMsg("aiResult", `🤖 AI Judge: ${match}% Match. ${feedback}`, "warning");
      } else {
        showMsg("aiResult", `🤖 AI Judge: ${match}% Match. ${feedback}`, "success");
      }
    } else {
      const errorMsg = data.detail || data.error || "AI evaluation failed due to server error.";
      showMsg("aiResult", `Error: ${errorMsg}`, "error");
    }
  } catch (err) {
    console.error(err);
    showMsg("aiResult", "Error connecting to AI service. Ensure server is running.", "error");
  } finally {
    setLoading('btn-ai', false);
  }
}

async function release() {
  if (!isEscrowActive || escrowAmount <= 0) {
    showMsg("paymentStatus", "No funds in escrow to release.", "error");
    return;
  }

  setLoading('btn-release', true);

  try {
    const tx = await escrowContract.approveMilestone(currentEscrowId, ethers.parseEther(escrowAmount.toString()));
    showMsg("paymentStatus", "Waiting for blockchain confirmation...", "success");
    await tx.wait();
    
    showMsg("paymentStatus", `💸 ₹${escrowAmount} released to freelancer wallet!`, "success");
    
    // Fetch updated reputation
    const repScore = await escrowContract.getReputation(userWallet);
    const newReputation = Number(repScore) / 10;
    
    // Animate reputation change
    const repEl = document.getElementById("rep");
    const progressFill = document.getElementById("rep-progress");
    
    repEl.style.transform = "scale(1.3)";
    setTimeout(() => { repEl.style.transform = "scale(1)"; }, 300);
    
    repEl.innerText = newReputation.toFixed(1);
    progressFill.style.width = `${(newReputation / 5) * 100}%`;
    
    // Reset state
    escrowAmount = 0;
    isEscrowActive = false;
    showMsg("escrowStatus", "Escrow is now empty.", "warning");
  } catch (error) {
    console.error(error);
    showMsg("paymentStatus", "Failed to release funds from smart contract.", "error");
  }
  
  setLoading('btn-release', false);
}