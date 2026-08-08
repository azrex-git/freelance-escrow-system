let escrowAmount = 0;
let reputation = 4.5;
let isEscrowActive = false;

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
      body: JSON.stringify({ amount: amount })
    });
    
    if (response.ok) {
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
  
  if (!isEscrowActive) {
    showMsg("aiResult", "No active escrow. Client must deposit first.", "warning");
    return;
  }
  
  if (!work.trim()) {
    showMsg("aiResult", "Please provide work details or a link.", "error");
    return;
  }

  setLoading('btn-ai', true);

  // Simulate AI evaluation delay
  setTimeout(() => {
    const len = work.length;
    if (len < 10) {
      showMsg("aiResult", "🤖 AI Judge: Quality too low. REJECTED. 0% Payment", "error");
    } else if (len < 30) {
      showMsg("aiResult", "🤖 AI Judge: Partial requirements met. 50% Match.", "warning");
    } else {
      showMsg("aiResult", "🤖 AI Judge: Exceptional Quality! 98% Match. APPROVED.", "success");
    }
    setLoading('btn-ai', false);
  }, 1500);
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