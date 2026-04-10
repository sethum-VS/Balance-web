// Balance Web — WebSocket Client
// Connects to the server's /ws endpoint and dispatches UI updates
// based on WSEvent messages from the Hub.

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
}

interface TimerStoppedPayload {
  sessionID: string;
  duration: number;
  creditsEarned: number;
}

interface BalanceUpdatedPayload {
  balance: number;
}

// State
let ws: WebSocket | null = null;
let clockInterval: ReturnType<typeof setInterval> | null = null;
let sessionStartTime: Date | null = null;
let activeCategory: string = "";

// DOM References (resolved after DOMContentLoaded)
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

function connectWebSocket(): void {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${protocol}//${window.location.host}/ws`;

  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log("[Balance WS] Connected");
  };

  ws.onmessage = (event: MessageEvent) => {
    try {
      const data: WSEvent = JSON.parse(event.data);
      console.log("[Balance WS] Event:", data.type, data.payload);
      dispatchEvent(data);
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

function dispatchEvent(event: WSEvent): void {
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
    default:
      console.warn("[Balance WS] Unknown event type:", event.type);
  }
}

function handleTimerStarted(payload: TimerStartedPayload): void {
  sessionStartTime = new Date(payload.startTime);
  activeCategory = payload.activityCategory;

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

  // Start the local clock interval
  startClockInterval();
}

function handleTimerStopped(_payload: TimerStoppedPayload): void {
  // Clear local clock
  stopClockInterval();
  sessionStartTime = null;

  // Reset clock display
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
}

function handleBalanceUpdated(payload: BalanceUpdatedPayload): void {
  if (balanceDisplay) {
    balanceDisplay.textContent = formatBalance(payload.balance);
  }
}

function startClockInterval(): void {
  stopClockInterval();
  clockInterval = setInterval(updateClock, 1000);
  updateClock(); // Immediate first tick
}

function stopClockInterval(): void {
  if (clockInterval !== null) {
    clearInterval(clockInterval);
    clockInterval = null;
  }
}

function updateClock(): void {
  if (!sessionStartTime || !clockDisplay) return;

  const now = new Date();
  const elapsed = Math.floor((now.getTime() - sessionStartTime.getTime()) / 1000);

  const hours = Math.floor(elapsed / 3600);
  const minutes = Math.floor((elapsed % 3600) / 60);
  const seconds = elapsed % 60;

  clockDisplay.textContent =
    pad(hours) + ":" + pad(minutes) + ":" + pad(seconds);

  // Update progress ring (complete one rotation per hour)
  if (progressCircle) {
    const circumference = 1162;
    const progress = (elapsed % 3600) / 3600;
    const offset = circumference - progress * circumference;
    progressCircle.style.strokeDashoffset = String(offset);

    // Color the ring based on category
    progressCircle.classList.remove("text-secondary", "text-error");
    progressCircle.classList.add(
      activeCategory === "toppingUp" ? "text-secondary" : "text-error"
    );
  }
}

function pad(n: number): string {
  return n < 10 ? "0" + n : String(n);
}

function formatBalance(n: number): string {
  return n.toLocaleString("en-US");
}

// Initialize on DOM ready
document.addEventListener("DOMContentLoaded", () => {
  console.log("[Balance] Initializing...");

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

  // Connect WebSocket
  connectWebSocket();
});
