// ==================== STATE ====================
const API_BASE_URL = 'https://freelance-escrow-system-backend.onrender.com';
let currentUser = 'client1';
let escrowsCache = [];
let planMilestones = [];

// ==================== INIT ====================
window.addEventListener('load', async () => {
  await refreshReputation();
  await refreshDashboard();
  populateDisputeDropdown();
});

// ==================== TAB NAVIGATION ====================
function switchTab(tabName) {
  document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));

  document.getElementById(`tab-${tabName}`).classList.add('active');
  document.querySelector(`.nav-btn[data-tab="${tabName}"]`).classList.add('active');

  if (tabName === 'dashboard') refreshDashboard();
  if (tabName === 'dispute') populateDisputeDropdown();
}

// ==================== HELPERS ====================
const showMsg = (id, msg, type = 'success') => {
  const el = document.getElementById(id);
  if (!el) return;
  el.innerText = msg;
  el.className = `status-msg show text-${type}`;
};

const setLoading = (btnId, loading) => {
  const btn = document.getElementById(btnId);
  if (!btn) return;
  const text = btn.querySelector('.btn-text');
  const loader = btn.querySelector('.loader');
  btn.disabled = loading;
  if (text) text.classList.toggle('hidden', loading);
  if (loader) loader.classList.toggle('hidden', !loading);
};

// ==================== REPUTATION ====================
async function refreshReputation() {
  try {
    const res = await fetch(API_BASE_URL + `/api/reputation/${currentUser}`);
    if (!res.ok) return;
    const data = await res.json();

    const repEl = document.getElementById('repScore');
    repEl.innerText = data.reputation.toFixed(1);
    repEl.style.transform = 'scale(1.15)';
    setTimeout(() => repEl.style.transform = 'scale(1)', 300);

    document.getElementById('repBarFill').style.width = `${(data.reputation / 5) * 100}%`;
    document.getElementById('statCompleted').innerText = data.completed_projects;
    document.getElementById('statDisputesWon').innerText = data.disputes_won;
    document.getElementById('statDisputesLost').innerText = data.disputes_lost;
    document.getElementById('statGhosting').innerText = data.ghosting_incidents;

    const badge = document.getElementById('trustBadge');
    badge.innerText = data.trust_tier;
    badge.setAttribute('data-tier', data.trust_tier);
  } catch (e) {
    console.error('Failed to fetch reputation', e);
  }
}

// ==================== DASHBOARD ====================
async function refreshDashboard() {
  try {
    const res = await fetch(API_BASE_URL + '/api/escrow/all');
    if (!res.ok) return;
    escrowsCache = await res.json();

    const container = document.getElementById('dashboardProjects');
    const empty = document.getElementById('dashboardEmpty');

    if (!escrowsCache || escrowsCache.length === 0) {
      empty.classList.remove('hidden');
      container.innerHTML = '';
      return;
    }

    empty.classList.add('hidden');
    container.innerHTML = escrowsCache.map(e => renderProjectCard(e)).join('');
  } catch (e) {
    console.error('Dashboard refresh failed', e);
  }
}

function renderProjectCard(escrow) {
  const statusClass = `status-${escrow.status}`;
  const completed = escrow.milestones.filter(m => m.status === 'released').length;
  const total = escrow.milestones.length;
  const progress = total > 0 ? Math.round((completed / total) * 100) : 0;

  let ghostingHtml = '';
  if (escrow.ghosting_risk && escrow.ghosting_risk !== 'none') {
    const icons = { low: '📋', medium: '🔶', high: '⚠️', critical: '🚨' };
    const labels = { low: 'Low ghosting risk', medium: 'Moderate ghosting risk', high: 'High ghosting risk!', critical: 'CRITICAL: Deadline missed!' };
    ghostingHtml = `<div class="ghosting-alert risk-${escrow.ghosting_risk}">${icons[escrow.ghosting_risk]} ${labels[escrow.ghosting_risk]}</div>`;
  }

  const milestonesHtml = escrow.milestones.map(m => {
    let scoreHtml = '';
    if (m.ai_match_score !== null && m.ai_match_score !== undefined) {
      const cls = m.ai_match_score >= 80 ? 'ai-score-high' : m.ai_match_score >= 50 ? 'ai-score-mid' : 'ai-score-low';
      scoreHtml = `<span class="ai-score-badge ${cls}">🤖 ${m.ai_match_score}%</span>`;
    }

    let deadlineMeta = '';
    if (m.deadline_at && m.status === 'pending') {
      const dl = new Date(m.deadline_at);
      const now = new Date();
      const hoursLeft = (dl - now) / (1000 * 60 * 60);
      if (hoursLeft < 0) {
        deadlineMeta = `<span style="color:#f87171">⏰ ${Math.abs(Math.round(hoursLeft))}h overdue</span>`;
      } else if (hoursLeft < 24) {
        deadlineMeta = `<span style="color:#fbbf24">⏰ ${Math.round(hoursLeft)}h left</span>`;
      } else {
        deadlineMeta = `<span>📅 ${Math.round(hoursLeft / 24)}d left</span>`;
      }
    }

    let actions = '';
    if (m.status === 'pending') {
      actions = `<button class="btn btn-secondary btn-sm" onclick="submitMilestone('${escrow.id}','${m.id}')">Submit Work</button>`;
    } else if (m.status === 'submitted') {
      actions = `
        <button class="btn btn-success btn-sm" onclick="approveMilestone('${escrow.id}','${m.id}')">✅ Approve</button>
        <button class="btn btn-danger btn-sm" onclick="rejectMilestone('${escrow.id}','${m.id}')">❌ Reject</button>
      `;
    } else if (m.status === 'approved') {
      actions = `<button class="btn btn-success btn-sm" onclick="releaseMilestone('${escrow.id}','${m.id}')">💸 Release Funds</button>`;
    } else if (m.status === 'released') {
      actions = `<span style="font-size:0.75rem;color:#34d399;">✅ Released</span>`;
    } else if (m.status === 'disputed') {
      actions = `<span style="font-size:0.75rem;color:#f87171;">⚖️ In Dispute</span>`;
    }

    return `
      <div class="milestone-row">
        <div class="m-indicator ${m.status}"></div>
        <div class="m-info">
          <div class="m-title">${m.description}</div>
          <div class="m-meta">${deadlineMeta} ${scoreHtml}</div>
        </div>
        <span class="m-amount-badge">₹${m.amount.toLocaleString()}</span>
        <div class="m-actions">${actions}</div>
      </div>
    `;
  }).join('');

  const auditCount = escrow.audit_logs ? escrow.audit_logs.length : 0;

  return `
    <div class="project-card">
      <div class="project-header">
        <span class="project-title">${escrow.project_name || 'Untitled Project'}</span>
        <span class="project-status ${statusClass}">${escrow.status}</span>
      </div>
      ${ghostingHtml}
      <div style="font-size:0.78rem; color:var(--text-muted); margin-bottom:0.8rem;">
        Total: ₹${escrow.total_amount.toLocaleString()} · ${total} milestones · ${progress}% complete · ${auditCount} audit logs
      </div>
      <div class="milestone-tracker">${milestonesHtml}</div>
    </div>
  `;
}

// ==================== MILESTONE ACTIONS ====================
let activeSubmitEscrow = null;
let activeSubmitMilestone = null;

function submitMilestone(escrowId, milestoneId) {
  activeSubmitEscrow = escrowId;
  activeSubmitMilestone = milestoneId;
  document.getElementById('submitFileName').value = '';
  document.getElementById('submitNote').value = '';
  document.getElementById('submitModal').classList.remove('hidden');
}

function closeSubmitModal() {
  document.getElementById('submitModal').classList.add('hidden');
  activeSubmitEscrow = null;
  activeSubmitMilestone = null;
}

document.getElementById('btn-submit-work').addEventListener('click', async () => {
  if (!activeSubmitEscrow || !activeSubmitMilestone) return;

  const fileName = document.getElementById('submitFileName').value.trim();
  const note = document.getElementById('submitNote').value.trim();

  if (!note && !fileName) {
    alert("Please provide some notes or a file name.");
    return;
  }

  let workPayload = note;
  if (fileName) {
    workPayload = `File Attached: [${fileName}]\n\nNotes:\n${note}`;
  }

  setLoading('btn-submit-work', true);
  try {
    const res = await fetch(API_BASE_URL + `/api/escrow/${activeSubmitEscrow}/milestone/${activeSubmitMilestone}/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ submitted_work: workPayload })
    });
    if (res.ok) {
      closeSubmitModal();
      await refreshDashboard();
    } else {
      alert("Failed to submit work");
    }
  } catch (e) {
    console.error(e);
  } finally {
    setLoading('btn-submit-work', false);
  }
});

async function approveMilestone(escrowId, milestoneId) {
  try {
    const btn = event.target;
    btn.disabled = true;
    btn.innerText = '⏳ Approving...';

    const res = await fetch(API_BASE_URL + `/api/escrow/${escrowId}/milestone/${milestoneId}/approve`, {
      method: 'POST'
    });

    if (res.ok) {
      await refreshDashboard();
    }
  } catch (e) {
    console.error(e);
  }
}

async function rejectMilestone(escrowId, milestoneId) {
  try {
    const btn = event.target;
    btn.disabled = true;
    btn.innerText = '⏳ Rejecting...';

    const res = await fetch(API_BASE_URL + `/api/escrow/${escrowId}/milestone/${milestoneId}/reject`, {
      method: 'POST'
    });

    if (res.ok) {
      alert("Milestone rejected! Please go to the Dispute Center to resolve this.");
      await refreshDashboard();
    }
  } catch (e) {
    console.error(e);
  }
}

async function releaseMilestone(escrowId, milestoneId) {
  try {
    const res = await fetch(API_BASE_URL + `/api/escrow/${escrowId}/milestone/${milestoneId}/release`, {
      method: 'POST'
    });
    if (res.ok) {
      await refreshDashboard();
      await refreshReputation();
    }
  } catch (e) {
    console.error(e);
  }
}

// ==================== AUTO PLAN ====================
async function autoPlan() {
  const idea = document.getElementById('projectIdea').value;
  const budget = parseFloat(document.getElementById('totalBudget').value);

  if (!idea || isNaN(budget) || budget <= 0) {
    showMsg('lockStatus', 'Enter project description and budget.', 'error');
    return;
  }

  setLoading('btn-autoplan', true);

  try {
    const res = await fetch(API_BASE_URL + '/api/escrow/auto-plan', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project_description: idea, total_budget: budget })
    });

    if (res.ok) {
      const data = await res.json();
      planMilestones = data.milestones || [];

      if (planMilestones.length > 0) {
        const resultEl = document.getElementById('planResult');
        resultEl.classList.remove('hidden');
        resultEl.innerHTML = `
          <div style="font-size:0.8rem; color:var(--accent); margin-bottom:0.6rem; font-weight:600;">
            ✨ AI Generated ${planMilestones.length} Milestones
          </div>
          ${planMilestones.map((m, i) => `
            <div class="milestone-item">
              <span class="m-desc">${i + 1}. ${m.description}</span>
              <span class="m-amount">₹${m.amount.toLocaleString()}</span>
              <span class="m-deadline">${m.deadline_days || 7}d</span>
            </div>
          `).join('')}
        `;

        // Show lock card
        document.getElementById('lockCard').classList.remove('hidden');
        document.getElementById('milestonesPreview').innerHTML = planMilestones.map((m, i) => `
          <div class="milestone-row" style="margin-bottom:0.4rem;">
            <div class="m-indicator pending"></div>
            <div class="m-info">
              <div class="m-title">${m.description}</div>
              <div class="m-meta">📅 ${m.deadline_days || 7} day deadline · ${m.client_instructions || 'No specific instructions'}</div>
            </div>
            <span class="m-amount-badge">₹${m.amount.toLocaleString()}</span>
          </div>
        `).join('');
      }
    } else {
      showMsg('lockStatus', 'AI planning failed. Try again.', 'error');
    }
  } catch (e) {
    showMsg('lockStatus', 'Error connecting to AI service.', 'error');
  } finally {
    setLoading('btn-autoplan', false);
  }
}

// ==================== LOCK ESCROW ====================
async function lockEscrow() {
  if (planMilestones.length === 0) return;

  const projectName = document.getElementById('projectName').value || 'Untitled Project';
  setLoading('btn-lock', true);

  try {
    const res = await fetch(API_BASE_URL + '/api/escrow', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        project_name: projectName,
        client_id: currentUser,
        freelancer_id: 'freelancer1',
        milestones: planMilestones.map(m => ({
          description: m.description,
          amount: m.amount,
          deadline_days: m.deadline_days || 7,
          client_instructions: m.client_instructions || m.description
        }))
      })
    });

    if (res.ok) {
      const data = await res.json();
      showMsg('lockStatus', `🔒 Escrow locked! ${data.milestones.length} milestones secured. ID: ${data.id.substring(0, 8)}...`, 'success');

      // Reset form
      planMilestones = [];
      document.getElementById('projectName').value = '';
      document.getElementById('projectIdea').value = '';
      document.getElementById('totalBudget').value = '';
      document.getElementById('planResult').classList.add('hidden');
      document.getElementById('lockCard').classList.add('hidden');

      // Switch to dashboard
      setTimeout(() => {
        switchTab('dashboard');
        populateDisputeDropdown();
      }, 1500);
    } else {
      showMsg('lockStatus', 'Failed to lock escrow.', 'error');
    }
  } catch (e) {
    showMsg('lockStatus', 'Server error.', 'error');
  } finally {
    setLoading('btn-lock', false);
  }
}

// ==================== DISPUTE ====================
function populateDisputeDropdown() {
  const select = document.getElementById('disputeEscrowSelect');
  if (!select) return;
  select.innerHTML = '<option value="">— Select an active escrow —</option>';
  escrowsCache.forEach(e => {
    if (e.status === 'active' || e.status === 'disputed') {
      select.innerHTML += `<option value="${e.id}">${e.project_name || e.id.substring(0, 8)} — ₹${e.total_amount.toLocaleString()}</option>`;
    }
  });
}

function loadDisputeMilestones() {
  const escrowId = document.getElementById('disputeEscrowSelect').value;
  const mSelect = document.getElementById('disputeMilestoneSelect');
  mSelect.innerHTML = '<option value="">— Select a milestone —</option>';

  if (!escrowId) return;
  const escrow = escrowsCache.find(e => e.id === escrowId);
  if (!escrow) return;

  escrow.milestones.forEach(m => {
    if (m.status !== 'released') {
      mSelect.innerHTML += `<option value="${m.id}">${m.description} — ₹${m.amount.toLocaleString()}</option>`;
    }
  });
}

async function raiseDispute() {
  const escrowId = document.getElementById('disputeEscrowSelect').value;
  const milestoneId = document.getElementById('disputeMilestoneSelect').value;
  const complaint = document.getElementById('clientComplaint').value;
  const defense = document.getElementById('freelancerDefense').value;
  const timeline = document.getElementById('timelineContext').value;
  const commLogs = document.getElementById('commLogs').value;

  if (!escrowId || !milestoneId) {
    showMsg('disputeStatus', 'Select project and milestone.', 'error');
    return;
  }
  if (!complaint || !defense) {
    showMsg('disputeStatus', 'Both complaint and defense required.', 'error');
    return;
  }

  setLoading('btn-dispute', true);

  try {
    const escrow = escrowsCache.find(e => e.id === escrowId);
    const milestone = escrow?.milestones.find(m => m.id === milestoneId);

    const res = await fetch(API_BASE_URL + `/api/escrow/${escrowId}/dispute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        milestone_id: milestoneId,
        client_complaint: complaint,
        freelancer_defense: defense,
        deliverables_text: milestone?.submitted_work || 'No deliverables submitted',
        communication_logs: commLogs || 'No logs provided',
        delivery_timeline: timeline || 'Not specified'
      })
    });

    const data = await res.json();

    if (res.ok) {
      showMsg('disputeStatus', 'AI Arbiter has rendered a verdict.', 'success');
      renderVerdict(data, escrow);
    } else {
      showMsg('disputeStatus', data.error || 'Dispute failed.', 'error');
    }
  } catch (e) {
    showMsg('disputeStatus', 'Server error.', 'error');
  } finally {
    setLoading('btn-dispute', false);
  }
}

function renderVerdict(data, escrow) {
  const card = document.getElementById('verdictCard');
  card.classList.remove('hidden');

  // Confidence ring
  const conf = data.confidence_score || 0;
  document.getElementById('ringText').innerText = `${conf}%`;
  const circumference = 2 * Math.PI * 45; // r=45
  const offset = circumference - (conf / 100) * circumference;
  document.getElementById('ringFill').style.strokeDashoffset = offset;

  // Ghosting
  const ghostingEl = document.getElementById('verdictGhosting');
  if (data.is_ghosting_detected) {
    ghostingEl.classList.remove('hidden');
    ghostingEl.innerText = `👻 Ghosting detected — ${data.ghosting_party} party`;
  } else {
    ghostingEl.classList.add('hidden');
  }

  // Split bar
  const cRefund = data.client_refund || 0;
  const fPayout = data.freelancer_payout || 0;
  const total = cRefund + fPayout;
  const cPct = total > 0 ? (cRefund / total) * 100 : 50;
  const fPct = total > 0 ? (fPayout / total) * 100 : 50;

  document.getElementById('splitBarClient').style.width = `${cPct}%`;
  document.getElementById('splitBarFreelancer').style.width = `${fPct}%`;
  document.getElementById('splitClientAmt').innerText = `Client: ₹${cRefund.toFixed(0)}`;
  document.getElementById('splitFreelancerAmt').innerText = `Freelancer: ₹${fPayout.toFixed(0)}`;

  // Reasoning
  document.getElementById('verdictReasoning').innerText = `"${data.reasoning || 'No reasoning provided.'}"`;

  // Evidence
  const evidence = data.evidence_summary || [];
  document.getElementById('evidenceCards').innerHTML = evidence.map(e => `
    <div class="evidence-card">
      <span class="evidence-icon">📌</span>
      <span>${e}</span>
    </div>
  `).join('');

  // Behavioral
  const beh = data.behavioral_analysis || {};
  const clientProf = beh.client_professionalism_score || beh.client_professionalism || 50;
  const freeProf = beh.freelancer_professionalism_score || beh.freelancer_professionalism || 50;
  document.getElementById('behClientBar').style.width = `${clientProf}%`;
  document.getElementById('behClientScore').innerText = clientProf;
  document.getElementById('behFreelancerBar').style.width = `${freeProf}%`;
  document.getElementById('behFreelancerScore').innerText = freeProf;
  document.getElementById('toneSummary').innerText = beh.tone_summary || '';

  // Store for execution
  window._disputeEscrowId = document.getElementById('disputeEscrowSelect').value;
}

async function executeDispute() {
  setLoading('btn-execute-dispute', true);

  try {
    showMsg('disputeStatus', '✅ Dispute verdict executed. Funds dispersed and reputations updated.', 'success');

    await refreshDashboard();
    await refreshReputation();
    populateDisputeDropdown();
  } catch (e) {
    showMsg('disputeStatus', 'Execution failed.', 'error');
  } finally {
    setLoading('btn-execute-dispute', false);
  }
}