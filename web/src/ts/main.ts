// Balance Web — Dual-Clock WebSocket Client
// Implements two independent clocks:
//   1. Global CR Balance — ticks up (toppingUp) or down (consuming) each second
//   2. Local Session Clock — pure elapsed time display (00:00:00)
// The server sends baseBalance with TIMER_STARTED so we can animate locally,
// then corrects any drift with an authoritative BALANCE_UPDATED on stop.

interface WSEvent {
  type: string;
  payload: any;
}

interface TimerStartedPayload {
  sessionID: string;
  activityID: string;
  activityName: string;
  activityCategory: string;
  startTime: string;
  baseBalance: number;
}

interface TimerStoppedPayload {
  sessionID: string;
  duration: number;
  creditsEarned: number;
}

interface BalanceUpdatedPayload {
  balance: number;
}

interface MobileStatusPayload {
  isOnline: boolean;
}

interface ActiveTimerResponse {
  active: boolean;
  sessionID?: string;
  activityID?: string;
  activityName?: string;
  activityCategory?: string;
  startTime?: string;
  baseBalance?: number;
}

// ──────────────────────────── State ────────────────────────────
let ws: WebSocket | null = null;
let clockInterval: ReturnType<typeof setInterval> | null = null;

// Dual-clock state
let sessionStartTime: Date | null = null;
let activeCategory: string = "";
let baseBalance: number = 0;       // CR pool snapshot at session start
let globalBalance: number = 0;     // Live-ticking global CR
let currentSessionTime: number = 0; // Local elapsed seconds

// ──────────────────────────── DOM Refs ────────────────────────────
let clockDisplay: HTMLElement | null;
let balanceDisplay: HTMLElement | null;
let sessionLabel: HTMLElement | null;
let sessionStatus: HTMLElement | null;
let statusPulse: HTMLElement | null;
let statusText: HTMLElement | null;
let progressCircle: SVGCircleElement | null;
let btnTopup: HTMLElement | null;
let btnStopTopup: HTMLElement | null;
let btnConsume: HTMLElement | null;
let btnStopConsume: HTMLElement | null;

// ──────────────────────────── WebSocket ────────────────────────────
function connectWebSocket(): void {
  const envWsUrl = document.body.dataset.wsUrl;

  let wsUrl: string;
  if (envWsUrl && envWsUrl !== "auto") {
    wsUrl = envWsUrl;
  } else {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    wsUrl = `${protocol}//${window.location.host}/ws`;
  }

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log("[Balance WS] Connected");
    void syncActiveSessionFromServer();
  };

  ws.onmessage = (event: MessageEvent) => {
    try {
      const data: WSEvent = JSON.parse(event.data);
      console.log("[Balance WS] Event:", data.type, data.payload);
      dispatchWSEvent(data);
    } catch (e) {
      console.error("[Balance WS] Failed to parse message:", e);
    }
  };

  ws.onclose = () => {
    console.log("[Balance WS] Disconnected. Reconnecting in 3s...");
    setTimeout(connectWebSocket, 3000);
  };

  ws.onerror = (err) => {
    console.error("[Balance WS] Error:", err);
    ws?.close();
  };
}

function dispatchWSEvent(event: WSEvent): void {
  switch (event.type) {
    case "TIMER_STARTED":
      handleTimerStarted(event.payload as TimerStartedPayload);
      break;
    case "TIMER_STOPPED":
      handleTimerStopped(event.payload as TimerStoppedPayload);
      break;
    case "BALANCE_UPDATED":
      handleBalanceUpdated(event.payload as BalanceUpdatedPayload);
      break;
    case "MOBILE_STATUS":
      handleMobileStatus(event.payload as MobileStatusPayload);
      break;
    default:
      console.warn("[Balance WS] Unknown event type:", event.type);
  }
}

async function syncActiveSessionFromServer(): Promise<void> {
  try {
    const response = await fetch("/api/timer/active", { method: "GET" });
    if (!response.ok) {
      return;
    }

    const payload = (await response.json()) as ActiveTimerResponse;
    if (payload.active && payload.startTime && payload.sessionID && payload.activityID) {
      handleTimerStarted({
        sessionID: payload.sessionID,
        activityID: payload.activityID,
        activityName: payload.activityName ?? "",
        activityCategory: payload.activityCategory ?? "",
        startTime: payload.startTime,
        baseBalance: payload.baseBalance ?? globalBalance,
      });
      return;
    }

    if (!payload.active && sessionStartTime) {
      handleTimerStopped({
        sessionID: "",
        duration: currentSessionTime,
        creditsEarned: 0,
      });
    }
  } catch (err) {
    console.warn("[Balance WS] Failed to sync active timer:", err);
  }
}

// ──────────────────────────── TIMER_STARTED ────────────────────────────
function handleTimerStarted(payload: TimerStartedPayload): void {
  // Save dual-clock state
  sessionStartTime = new Date(payload.startTime);
  activeCategory = payload.activityCategory;
  baseBalance = payload.baseBalance;
  globalBalance = baseBalance;
  currentSessionTime = 0;

  // Update session label
  if (sessionLabel) {
    sessionLabel.textContent = "Current Session";
  }

  // Update status indicator
  if (sessionStatus) {
    sessionStatus.className =
      activeCategory === "toppingUp"
        ? "flex items-center gap-3 mt-6 bg-secondary/10 px-6 py-2 rounded-full border border-secondary/20"
        : "flex items-center gap-3 mt-6 bg-error/10 px-6 py-2 rounded-full border border-error/20";
  }

  if (statusPulse) {
    statusPulse.className =
      activeCategory === "toppingUp"
        ? "w-2 h-2 rounded-full bg-secondary animate-pulse"
        : "w-2 h-2 rounded-full bg-error animate-pulse";
  }

  if (statusText) {
    const label = activeCategory === "toppingUp" ? "Topping Up" : "Consuming";
    statusText.textContent = `${label} — ${payload.activityName}`;
    statusText.className =
      activeCategory === "toppingUp"
        ? "text-secondary font-bold tracking-wide"
        : "text-error font-bold tracking-wide";
  }

  // Show stop button, hide start button for the active category
  if (activeCategory === "toppingUp") {
    if (btnTopup) btnTopup.classList.add("hidden");
    if (btnStopTopup) btnStopTopup.classList.remove("hidden");
  } else {
    if (btnConsume) btnConsume.classList.add("hidden");
    if (btnStopConsume) btnStopConsume.classList.remove("hidden");
  }

  // Start the dual-clock interval
  startClockInterval();
}

// ──────────────────────────── TIMER_STOPPED ────────────────────────────
function handleTimerStopped(_payload: TimerStoppedPayload): void {
  // Invalidate timer
  stopClockInterval();
  sessionStartTime = null;
  currentSessionTime = 0;

  // Reset session clock display
  if (clockDisplay) {
    clockDisplay.textContent = "00:00:00";
  }

  // Reset session status to idle
  if (sessionLabel) {
    sessionLabel.textContent = "No Active Session";
  }

  if (sessionStatus) {
    sessionStatus.className =
      "flex items-center gap-3 mt-6 bg-surface-container-high/50 px-6 py-2 rounded-full border border-outline-variant/20 opacity-50";
  }

  if (statusPulse) {
    statusPulse.className = "w-2 h-2 rounded-full bg-outline";
  }

  if (statusText) {
    statusText.textContent = "Idle";
    statusText.className = "text-on-surface-variant font-bold tracking-wide";
  }

  // Reset progress ring
  if (progressCircle) {
    progressCircle.style.strokeDashoffset = "1162";
  }

  // Reset buttons
  if (btnTopup) btnTopup.classList.remove("hidden");
  if (btnStopTopup) btnStopTopup.classList.add("hidden");
  if (btnConsume) btnConsume.classList.remove("hidden");
  if (btnStopConsume) btnStopConsume.classList.add("hidden");

  activeCategory = "";
  // Note: globalBalance is NOT reset here — it will be corrected
  // by the BALANCE_UPDATED event that follows immediately
}

// ──────────────────────────── BALANCE_UPDATED ────────────────────────────
function handleBalanceUpdated(payload: BalanceUpdatedPayload): void {
  // Server sends the authoritative final balance — corrects any local drift
  globalBalance = payload.balance;
  baseBalance = payload.balance;

  if (balanceDisplay) {
    balanceDisplay.textContent = formatBalance(globalBalance);
  }
}

// ──────────────────────────── MOBILE_STATUS ────────────────────────────
function handleMobileStatus(payload: MobileStatusPayload): void {
  const overlay = document.getElementById("mobile-offline-overlay");
  const lockableUIs = document.querySelectorAll(".lockable-ui");

  if (payload.isOnline) {
    overlay?.classList.add("hidden");
    lockableUIs.forEach((el) => {
      (el as HTMLElement).style.pointerEvents = "auto";
      (el as HTMLElement).style.opacity = "1";
    });
  } else {
    overlay?.classList.remove("hidden");
    lockableUIs.forEach((el) => {
      (el as HTMLElement).style.pointerEvents = "none";
      (el as HTMLElement).style.opacity = "0.5";
    });
  }
}

// ──────────────────────────── Dual-Clock Tick ────────────────────────────
function startClockInterval(): void {
  stopClockInterval();
  clockInterval = setInterval(tick, 1000);
  tick(); // Immediate first tick
}

function stopClockInterval(): void {
  if (clockInterval !== null) {
    clearInterval(clockInterval);
    clockInterval = null;
  }
}

function tick(): void {
  if (!sessionStartTime) return;

  const now = new Date();
  const elapsed = Math.floor(
    (now.getTime() - sessionStartTime.getTime()) / 1000
  );

  // ── Clock 1: Local Session Timer ──
  currentSessionTime = elapsed;

  if (clockDisplay) {
    const hours = Math.floor(elapsed / 3600);
    const minutes = Math.floor((elapsed % 3600) / 60);
    const seconds = elapsed % 60;
    clockDisplay.textContent = pad(hours) + ":" + pad(minutes) + ":" + pad(seconds);
  }

  // ── Clock 2: Global CR Balance ──
  if (activeCategory === "toppingUp") {
    globalBalance = baseBalance + elapsed;
  } else if (activeCategory === "consuming") {
    globalBalance = baseBalance - elapsed;
    if (globalBalance <= 0) { globalBalance = 0; }
  }

  if (balanceDisplay) {
    balanceDisplay.textContent = formatBalance(globalBalance);
  }

  // ── Progress Ring ──
  if (progressCircle) {
    const circumference = 1162;
    const progress = (elapsed % 3600) / 3600;
    const offset = circumference - progress * circumference;
    progressCircle.style.strokeDashoffset = String(offset);

    progressCircle.classList.remove("text-secondary", "text-error");
    progressCircle.classList.add(
      activeCategory === "toppingUp" ? "text-secondary" : "text-error"
    );
  }
}

// ──────────────────────────── Helpers ────────────────────────────
function pad(n: number): string {
  return n < 10 ? "0" + n : String(n);
}

function formatBalance(n: number): string {
  return n.toLocaleString("en-US");
}

// ──────────────────────────── Init ────────────────────────────
document.addEventListener("DOMContentLoaded", () => {
  console.log("[Balance] Dual-Clock Architecture Initialized");

  // Resolve DOM references
  clockDisplay = document.getElementById("clock-display");
  balanceDisplay = document.getElementById("balance-display");
  sessionLabel = document.getElementById("session-label");
  sessionStatus = document.getElementById("session-status");
  statusPulse = document.getElementById("status-pulse");
  statusText = document.getElementById("status-text");
  progressCircle = document.getElementById("progress-circle") as SVGCircleElement | null;
  btnTopup = document.getElementById("btn-topup");
  btnStopTopup = document.getElementById("btn-stop-topup");
  btnConsume = document.getElementById("btn-consume");
  btnStopConsume = document.getElementById("btn-stop-consume");

  // Read initial balance from the DOM (server-rendered)
  if (balanceDisplay) {
    const initial = parseInt(balanceDisplay.textContent?.replace(/,/g, "") || "0", 10);
    globalBalance = isNaN(initial) ? 0 : initial;
    baseBalance = globalBalance;
  }

  // Handle initial offline state
  const overlay = document.getElementById("mobile-offline-overlay");
  if (overlay && !overlay.classList.contains("hidden")) {
    handleMobileStatus({ isOnline: false });
  }

  // Handle HTMX Errors
  document.body.addEventListener("showError", function (evt: any) {
    alert(evt.detail.value);
  });

  void syncActiveSessionFromServer();
  connectWebSocket();
});
