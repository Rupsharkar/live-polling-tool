import { useState } from "react";

function App() {
  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState(["", ""]);
  const [created, setCreated] = useState(false);
  const [votes, setVotes] = useState([0, 0]);
  const [voted, setVoted] = useState(false);

  function addOption() {
    if (options.length < 5) {
      setOptions([...options, ""]);
      setVotes([...votes, 0]);
    }
  }

  function updateOption(index, value) {
    const updated = [...options];
    updated[index] = value;
    setOptions(updated);
  }

  function createPoll(e) {
    e.preventDefault();

    if (!question.trim()) {
      alert("Enter a question");
      return;
    }

    if (options.some((option) => !option.trim())) {
      alert("Please fill all options");
      return;
    }

    setVotes(options.map(() => 0));
    setVoted(false);
    setCreated(true);
  }

  function vote(index) {
    if (voted) return;

    const updated = [...votes];
    updated[index]++;
    setVotes(updated);
    setVoted(true);
  }

  function resetPoll() {
    setQuestion("");
    setOptions(["", ""]);
    setVotes([0, 0]);
    setVoted(false);
    setCreated(false);
  }

  const totalVotes = votes.reduce((sum, vote) => sum + vote, 0);

  return (
    <div
      style={{
        minHeight: "100vh",
        background: "#f5f7fb",
        fontFamily: "Arial",
        padding: "40px",
      }}
    >
      <div
        style={{
          maxWidth: "700px",
          margin: "auto",
          background: "white",
          padding: "35px",
          borderRadius: "16px",
          boxShadow: "0 5px 25px rgba(0,0,0,0.1)",
        }}
      >
        <h1>PulsePoll</h1>

        {!created ? (
          <>
            <p>Create a poll and share it with your audience.</p>

            <form onSubmit={createPoll}>
              <label>
                <strong>Question</strong>
              </label>

              <input
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="What is your favorite language?"
                style={{
                  display: "block",
                  width: "100%",
                  padding: "12px",
                  margin: "10px 0 20px",
                  boxSizing: "border-box",
                }}
              />

              <h3>Options</h3>

              {options.map((option, index) => (
                <input
                  key={index}
                  value={option}
                  onChange={(e) =>
                    updateOption(index, e.target.value)
                  }
                  placeholder={`Option ${index + 1}`}
                  style={{
                    display: "block",
                    width: "100%",
                    padding: "12px",
                    margin: "10px 0",
                    boxSizing: "border-box",
                  }}
                />
              ))}

              {options.length < 5 && (
                <button
                  type="button"
                  onClick={addOption}
                  style={{
                    padding: "10px 15px",
                    marginTop: "10px",
                  }}
                >
                  + Add Option
                </button>
              )}

              <br />
              <br />

              <button
                type="submit"
                style={{
                  padding: "12px 25px",
                  background: "#111827",
                  color: "white",
                  border: "none",
                  borderRadius: "8px",
                  cursor: "pointer",
                }}
              >
                Create Poll
              </button>
            </form>
          </>
        ) : (
          <>
            <p>🔴 LIVE POLL</p>

            <h2>{question}</h2>

            <p>
              <strong>{totalVotes}</strong>{" "}
              {totalVotes === 1 ? "vote" : "votes"}
            </p>

            {options.map((option, index) => {
              const percentage =
                totalVotes === 0
                  ? 0
                  : Math.round((votes[index] / totalVotes) * 100);

              return (
                <div key={index} style={{ marginBottom: "18px" }}>
                  <button
                    onClick={() => vote(index)}
                    disabled={voted}
                    style={{
                      width: "100%",
                      padding: "14px",
                      textAlign: "left",
                      border: "1px solid #ddd",
                      borderRadius: "8px",
                      background: "white",
                      cursor: voted ? "default" : "pointer",
                      fontSize: "16px",
                    }}
                  >
                    <strong>{option}</strong>

                    <span style={{ float: "right" }}>
                      {percentage}%
                    </span>
                  </button>

                  <div
                    style={{
                      height: "8px",
                      background: "#eee",
                      borderRadius: "10px",
                      marginTop: "5px",
                    }}
                  >
                    <div
                      style={{
                        width: `${percentage}%`,
                        height: "100%",
                        background: "#111827",
                        borderRadius: "10px",
                      }}
                    />
                  </div>

                  <small>{votes[index]} votes</small>
                </div>
              );
            })}

            {voted && (
              <p style={{ color: "green" }}>
                ✅ Thanks! Your vote has been recorded.
              </p>
            )}

            <button
              onClick={resetPoll}
              style={{
                padding: "10px 20px",
                marginTop: "15px",
              }}
            >
              Create Another Poll
            </button>
          </>
        )}
      </div>
    </div>
  );
}

export default App;