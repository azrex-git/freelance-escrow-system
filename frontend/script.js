let escrowAmount = 0;
let reputation = 4.5;
let isEscrowActive = false;
let currentEscrowId = "";
let currentMilestoneId = "";

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
  
  if (isNaN(amount) || amount <= 0) {
    showMsg("escrowStatus", "Please enter a valid amount.", "error");
    return;
  }

  setLoading('btn-deposit', true);
  
  try {
    // API Call to Backend
    const response = await fetch('/api/escrow', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        client_id: "client1",
        freelancer_id: "freelancer1",
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
      
      // Simulate real world wait
      setTimeout(() => {
        escrowAmount = amount;
        isEscrowActive = true;
        showMsg("escrowStatus", `✅ ₹${escrowAmount} locked in secure smart contract.`, "success");
        amountInput.value = '';
        setLoading('btn-deposit', false);
      }, 800);
    } else {
      showMsg("escrowStatus", "Failed to create escrow.", "error");
      setLoading('btn-deposit', false);
    }
  } catch (err) {
    console.error(err);
    // Fallback simulation if backend fetch fails
    setTimeout(() => {
      escrowAmount = amount;
      isEscrowActive = true;
      showMsg("escrowStatus", `✅ ₹${escrowAmount} locked in secure smart contract (Simulated).`, "success");
      amountInput.value = '';
      setLoading('btn-deposit', false);
    }, 1000);
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

  // Simulate Blockchain Transaction
  setTimeout(() => {
    showMsg("paymentStatus", `💸 ₹${escrowAmount} released to freelancer wallet!`, "success");
    
    // Update Reputation
    reputation += 0.1;
    if (reputation > 5.0) reputation = 5.0;
    
    // Animate reputation change
    const repEl = document.getElementById("rep");
    const progressFill = document.getElementById("rep-progress");
    
    repEl.style.transform = "scale(1.3)";
    setTimeout(() => { repEl.style.transform = "scale(1)"; }, 300);
    
    repEl.innerText = reputation.toFixed(1);
    progressFill.style.width = `${(reputation / 5) * 100}%`;
    
    // Reset state
    escrowAmount = 0;
    isEscrowActive = false;
    showMsg("escrowStatus", "Escrow is now empty.", "warning");
    
    setLoading('btn-release', false);
  }, 2000);
}