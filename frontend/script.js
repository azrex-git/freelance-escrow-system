let escrowAmount = 0;
let reputation = 4.5;

function deposit() {
  escrowAmount = document.getElementById("amount").value;
  let status = document.getElementById("escrowStatus");
  if(escrowAmount > 0) {
    status.innerText = "✅ ₹" + escrowAmount + " Locked in Smart Contract!";
    status.style.color = "#FFD700";
  } else {
    alert("Amount dao bhai!");
  }
}

function checkAI() {
  let work = document.getElementById("work").value;
  let resultBox = document.getElementById("aiResult");
  
  if(work.length < 5) {
    resultBox.innerText = "🤖 AI Judge: Kaaj khub choto. REJECT. 0% Payment";
    resultBox.style.color = "#ff4d4d";
  } else if(work.length < 20) {
    resultBox.innerText = "🤖 AI Judge: 50% Match. PARTIAL. 50% Payment";
    resultBox.style.color = "#ffa500";
  } else {
    resultBox.innerText = "🤖 AI Judge: 95% Match. APPROVE. 100% Payment";
    resultBox.style.color = "#FFD700";
  }
}

function release() {
  let payStatus = document.getElementById("paymentStatus");
  if(escrowAmount > 0) {
    let payment = escrowAmount;
    payStatus.innerText = "💰 ₹" + payment + " Auto Sent to Freelancer Wallet!";
    payStatus.style.color = "#FFD700";
    
    reputation = reputation + 0.2;
    if(reputation > 5.0) reputation = 5.0;
    document.getElementById("rep").innerText = reputation.toFixed(1);
    document.getElementById("escrowStatus").innerText = "Escrow Khali";
    escrowAmount = 0;
  } else {
    alert("Age Escrow te taka joma koro!");
  }
}