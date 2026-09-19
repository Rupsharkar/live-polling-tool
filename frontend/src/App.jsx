import { useEffect, useState } from "react";

const API_URL =
  "https://live-polling-tool-s4wv.onrender.com/api";

const WS_URL =
  "wss://live-polling-tool-s4wv.onrender.com";

function App() {
  const [page, setPage] = useState(
    window.location.pathname.startsWith("/poll/")
      ? "poll"
      : "login"
  );

  const [token, setToken] = useState(
    localStorage.getItem("pulse_token") || ""
  );

  const [authMode, setAuthMode] = useState("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState(["", ""]);
  const [pollId, setPollId] = useState("");
  const [poll, setPoll] = useState(null);
  const [counts, setCounts] = useState([]);
  const [voted, setVoted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [copied, setCopied] = useState(false);

  const isLoggedIn = Boolean(token);

  function getPollIdFromUrl() {
    return window.location.pathname.split("/poll/")[1];
  }

  function goTo(path) {
    window.history.pushState({}, "", path);
    if (path.startsWith("/poll/")) {
      setPage("poll");
    } else if (path === "/create") {
      setPage("create");
    } else {
      setPage("login");
    }
  }

  useEffect(() => {
    const handlePopState = () => {
      const path = window.location.pathname;
      setPage(
        path.startsWith("/poll/")
          ? "poll"
          : path === "/create"
          ? "create"
          : "login"
      );
    };

    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, []);

  useEffect(() => {
    if (page !== "poll") return;

    const id = getPollIdFromUrl();
    if (!id) return;

    setPollId(id);
    loadPoll(id);
  }, [page]);

  useEffect(() => {
    if (!pollId) return;

    const socket = new WebSocket(
      `${WS_URL}/ws/polls/${pollId}`
    );

    socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.counts) {
          setCounts(data.counts);
        }
      } catch {
        console.log("Invalid WebSocket message");
      }
    };

    socket.onerror = () => {
      console.log("WebSocket connection error");
    };

    return () => {
      socket.close();
    };
  }, [pollId]);

  async function loadPoll(id) {
    try {
      setLoading(true);

      const response = await fetch(
        `${API_URL}/polls/${id}`
      );

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Poll not found");
        return;
      }

      setPoll(data);
      setCounts(data.counts || []);
    } catch {
      setMessage("Unable to connect to the server.");
    } finally {
      setLoading(false);
    }
  }

  async function handleAuth(e) {
    e.preventDefault();

    if (!email.trim() || !password.trim()) {
      setMessage("Please enter email and password.");
      return;
    }

    try {
      setLoading(true);
      setMessage("");

      const endpoint =
        authMode === "login"
          ? "/auth/login"
          : "/auth/signup";

      const response = await fetch(
        `${API_URL}${endpoint}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            email,
            password,
          }),
        }
      );

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Authentication failed.");
        return;
      }

      localStorage.setItem("pulse_token", data.token);
      setToken(data.token);
      setEmail("");
      setPassword("");
      setMessage("");
      setPage("create");
    } catch (error) {
      console.error("Authentication error:", error);
      setMessage("Unable to connect to the backend.");
    } finally {
      setLoading(false);
    }
  }

  function logout() {
    localStorage.removeItem("pulse_token");
    setToken("");
    setPage("login");
    setMessage("");
  }

  function addOption() {
    if (options.length < 5) {
      setOptions([...options, ""]);
    }
  }

  function updateOption(index, value) {
    const updated = [...options];
    updated[index] = value;
    setOptions(updated);
  }

  async function createPoll(e) {
    e.preventDefault();

    if (!question.trim()) {
      setMessage("Enter a poll question.");
      return;
    }

    if (options.some((option) => !option.trim())) {
      setMessage("Please fill all options.");
      return;
    }

    try {
      setLoading(true);
      setMessage("");

      const response = await fetch(
        `${API_URL}/polls`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            question: question.trim(),
            options: options.map((option) => option.trim()),
          }),
        }
      );

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Could not create poll.");
        return;
      }

      setQuestion("");
      setOptions(["", ""]);
      goTo(`/poll/${data.id}`);
    } catch {
      setMessage("Unable to connect to the backend.");
    } finally {
      setLoading(false);
    }
  }

  async function vote(index) {
    if (voted || !poll?.open) return;

    try {
      setLoading(true);
      setMessage("");

      let voterId = localStorage.getItem("pulse_voter_id");

      if (!voterId) {
        voterId = crypto.randomUUID();
        localStorage.setItem("pulse_voter_id", voterId);
      }

      const response = await fetch(
        `${API_URL}/polls/${pollId}/vote`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            option: index,
            voterId,
          }),
        }
      );

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Could not record vote.");
        return;
      }

      setCounts(data.counts || []);
      setVoted(true);
    } catch {
      setMessage("Unable to submit your vote.");
    } finally {
      setLoading(false);
    }
  }

  async function closePoll() {
    if (!token || !pollId) return;

    try {
      const response = await fetch(
        `${API_URL}/polls/${pollId}/close`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      const data = await response.json();

      if (!response.ok) {
        setMessage(data.error || "Could not close poll.");
        return;
      }

      setPoll({
        ...poll,
        open: false,
      });

      setMessage("Poll closed successfully.");
    } catch {
      setMessage("Unable to close poll.");
    }
  }

  async function copyLink() {
    await navigator.clipboard.writeText(window.location.href);
    setCopied(true);

    setTimeout(() => {
      setCopied(false);
    }, 2000);
  }

  function totalVotes() {
    return counts.reduce(
      (total, count) => total + Number(count || 0),
      0
    );
  }

  function percentage(index) {
    const total = totalVotes();

    if (!total) return 0;

    return Math.round(
      (Number(counts[index] || 0) / total) * 100
    );
  }

 if (page === "login") {
    return (
      <div style={styles.page}>
        <div style={styles.card}>
          <div style={styles.logo}>PulsePoll</div>

          <div style={styles.badge}>LIVE POLLING</div>

          <h1 style={styles.title}>
            {authMode === "login"
              ? "Welcome back"
              : "Create your account"}
          </h1>

          <p style={styles.subtitle}>
            {authMode === "login"
              ? "Sign in to create and manage your polls."
              : "Create an account and start polling instantly."}
          </p>

          <form onSubmit={handleAuth}>
            <label style={styles.label}>Email</label>

            <input
              style={styles.input}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
            />

            <label style={styles.label}>Password</label>

            <input
              style={styles.input}
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
            />

            {message && (
              <div style={styles.error}>{message}</div>
            )}

            <button
              style={styles.primaryButton}
              disabled={loading}
            >
              {loading
                ? "Please wait..."
                : authMode === "login"
                ? "Login"
                : "Sign Up"}
            </button>
          </form>

          <button
            style={styles.linkButton}
            onClick={() => {
              setAuthMode(
                authMode === "login" ? "signup" : "login"
              );
              setMessage("");
            }}
          >
            {authMode === "login"
              ? "Don't have an account? Sign up"
              : "Already have an account? Login"}
          </button>
        </div>
      </div>
    );
  }

  if (page === "create") {
    return (
      <div style={styles.page}>
        <div style={styles.card}>
          <div style={styles.topBar}>
            <div style={styles.logo}>PulsePoll</div>

            <button
              style={styles.smallButton}
              onClick={logout}
            >
              Logout
            </button>
          </div>

          <div style={styles.badge}>CREATE A POLL</div>

          <h1 style={styles.title}>
            Ask your audience.
            <br />
            Get live answers.
          </h1>

          <p style={styles.subtitle}>
            Create a poll and share the link with anyone.
            Results update live without refreshing.
          </p>

          <form onSubmit={createPoll}>
            <label style={styles.label}>Question</label>

            <input
              style={styles.input}
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="What is your favorite programming language?"
            />

            <label style={styles.label}>Options</label>

            {options.map((option, index) => (
              <input
                key={index}
                style={styles.input}
                value={option}
                onChange={(e) =>
                  updateOption(index, e.target.value)
                }
                placeholder={`Option ${index + 1}`}
              />
            ))}

            {options.length < 5 && (
              <button
                type="button"
                style={styles.secondaryButton}
                onClick={addOption}
              >
                + Add Option
              </button>
            )}

            {message && (
              <div style={styles.error}>{message}</div>
            )}

            <button
              type="submit"
              style={styles.primaryButton}
              disabled={loading}
            >
              {loading ? "Creating..." : "Create Poll"}
            </button>
          </form>
        </div>
      </div>
    );
  }

  if (page === "poll") {
    return (
      <div style={styles.page}>
        <div style={styles.card}>
          {loading && !poll ? (
            <p>Loading poll...</p>
          ) : poll ? (
            <>
              <div style={styles.topBar}>
                <div style={styles.logo}>PulsePoll</div>

                <div
                  style={{
                    ...styles.status,
                    background: poll.open
                      ? "#dcfce7"
                      : "#fee2e2",
                    color: poll.open
                      ? "#166534"
                      : "#991b1b",
                  }}
                >
                  {poll.open ? "● LIVE" : "● CLOSED"}
                </div>
              </div>

              <h1 style={styles.pollQuestion}>
                {poll.question}
              </h1>

              <div style={styles.voteCount}>
                <strong>{totalVotes()}</strong>{" "}
                {totalVotes() === 1 ? "vote" : "votes"}
              </div>

              {poll.options.map((option, index) => (
                <div
                  key={index}
                  style={styles.optionContainer}
                >
                  <button
                    onClick={() => vote(index)}
                    disabled={
                      voted || !poll.open || loading
                    }
                    style={{
                      ...styles.voteButton,
                      opacity:
                        voted || !poll.open ? 0.85 : 1,
                    }}
                  >
                    <strong>{option}</strong>

                    <span>{percentage(index)}%</span>
                  </button>

                  <div style={styles.progressBackground}>
                    <div
                      style={{
                        ...styles.progress,
                        width: `${percentage(index)}%`,
                      }}
                    />
                  </div>

                  <small>
                    {Number(counts[index] || 0)} votes
                  </small>
                </div>
              ))}

              {voted && (
                <div style={styles.success}>
                  ✓ Thanks! Your vote has been recorded.
                </div>
              )}

              {message && (
                <div style={styles.error}>{message}</div>
              )}

              <button
                style={styles.shareButton}
                onClick={copyLink}
              >
                {copied
                  ? "✓ Link Copied!"
                  : "🔗 Copy Share Link"}
              </button>

              {isLoggedIn && (
                <button
                  style={styles.closeButton}
                  onClick={closePoll}
                  disabled={!poll.open}
                >
                  {poll.open ? "Close Poll" : "Poll Closed"}
                </button>
              )}

              <button
                style={styles.linkButton}
                onClick={() => {
                  if (isLoggedIn) {
                    goTo("/create");
                    setPage("create");
                  } else {
                    goTo("/");
                    setPage("login");
                  }
                }}
              >
                Create another poll
              </button>
            </>
          ) : (
            <>
              <h1>Poll not found</h1>
              <p>{message}</p>
            </>
          )}
        </div>
      </div>
    );
  }

  return null;
}

const styles = {
  page: {
    minHeight: "100vh",
    background:
      "linear-gradient(135deg, #eef2ff 0%, #f8fafc 50%, #ecfeff 100%)",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    padding: "24px",
    fontFamily:
      "Inter, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif",
    boxSizing: "border-box",
  },

  card: {
    width: "100%",
    maxWidth: "620px",
    background: "#ffffff",
    borderRadius: "24px",
    padding: "40px",
    boxShadow:
      "0 20px 60px rgba(15, 23, 42, 0.12)",
    boxSizing: "border-box",
  },

  logo: {
    fontSize: "24px",
    fontWeight: "800",
    color: "#111827",
    letterSpacing: "-1px",
  },

  badge: {
    display: "inline-block",
    marginTop: "28px",
    marginBottom: "14px",
    padding: "7px 11px",
    borderRadius: "999px",
    background: "#eef2ff",
    color: "#4f46e5",
    fontSize: "11px",
    fontWeight: "800",
    letterSpacing: "1px",
  },

  title: {
    fontSize: "36px",
    lineHeight: "1.12",
    letterSpacing: "-1.5px",
    margin: "0 0 12px",
    color: "#111827",
  },

  subtitle: {
    color: "#64748b",
    lineHeight: "1.6",
    marginBottom: "30px",
  },

  topBar: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: "20px",
  },

  status: {
    padding: "7px 12px",
    borderRadius: "999px",
    fontSize: "12px",
    fontWeight: "700",
  },

  label: {
    display: "block",
    fontWeight: "700",
    color: "#334155",
    marginBottom: "8px",
    marginTop: "18px",
  },

  input: {
    width: "100%",
    padding: "14px 16px",
    border: "1px solid #dbe2ea",
    borderRadius: "12px",
    fontSize: "15px",
    boxSizing: "border-box",
    outline: "none",
    marginBottom: "4px",
  },

  primaryButton: {
    width: "100%",
    marginTop: "22px",
    padding: "15px",
    border: "none",
    borderRadius: "12px",
    background: "#111827",
    color: "white",
    fontSize: "15px",
    fontWeight: "700",
    cursor: "pointer",
  },

  secondaryButton: {
    marginTop: "10px",
    padding: "11px 15px",
    border: "1px solid #cbd5e1",
    borderRadius: "10px",
    background: "white",
    color: "#334155",
    fontWeight: "600",
    cursor: "pointer",
  },

  smallButton: {
    padding: "8px 13px",
    border: "1px solid #dbe2ea",
    borderRadius: "9px",
    background: "white",
    cursor: "pointer",
  },

  linkButton: {
    width: "100%",
    marginTop: "18px",
    border: "none",
    background: "transparent",
    color: "#4f46e5",
    fontWeight: "600",
    cursor: "pointer",
  },

  error: {
    marginTop: "16px",
    padding: "12px",
    background: "#fef2f2",
    color: "#b91c1c",
    borderRadius: "10px",
    fontSize: "14px",
  },

  success: {
    marginTop: "20px",
    padding: "13px",
    background: "#f0fdf4",
    color: "#166534",
    borderRadius: "10px",
    fontWeight: "600",
    fontSize: "14px",
  },

  pollQuestion: {
    fontSize: "30px",
    lineHeight: "1.2",
    marginTop: "30px",
    marginBottom: "8px",
    color: "#111827",
  },

  voteCount: {
    color: "#64748b",
    marginBottom: "28px",
  },

  optionContainer: {
    marginBottom: "20px",
  },

  voteButton: {
    width: "100%",
    padding: "15px 17px",
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    border: "1px solid #dbe2ea",
    borderRadius: "12px",
    background: "#ffffff",
    color: "#111827",
    fontSize: "15px",
    cursor: "pointer",
    boxSizing: "border-box",
  },

  progressBackground: {
    height: "8px",
    background: "#e5e7eb",
    borderRadius: "999px",
    marginTop: "7px",
    overflow: "hidden",
  },

  progress: {
    height: "100%",
    background: "#111827",
    borderRadius: "999px",
    transition: "width 0.3s ease",
  },

  shareButton: {
    width: "100%",
    padding: "13px",
    marginTop: "12px",
    border: "1px solid #cbd5e1",
    borderRadius: "11px",
    background: "white",
    fontWeight: "700",
    cursor: "pointer",
  },

  closeButton: {
    width: "100%",
    padding: "13px",
    marginTop: "10px",
    border: "none",
    borderRadius: "11px",
    background: "#fee2e2",
    color: "#991b1b",
    fontWeight: "700",
    cursor: "pointer",
  },
};

export default App;