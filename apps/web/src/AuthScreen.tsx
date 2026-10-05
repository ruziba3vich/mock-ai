import { FormEvent, useState } from "react";
import { authAPI, User } from "./auth";

type Props = { onAuthenticated: (user: User) => void };

export default function AuthScreen({ onAuthenticated }: Props) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const user = mode === "register"
        ? await authAPI.register(displayName, email, password)
        : await authAPI.login(email, password);
      onAuthenticated(user);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Could not authenticate.");
    } finally {
      setSubmitting(false);
    }
  }

  function switchMode() {
    setMode((current) => current === "login" ? "register" : "login");
    setError("");
  }

  return (
    <main className="auth-page">
      <section className="auth-intro">
        <a className="brand" href="#" aria-label="Mock AI home">
          <span className="brand-mark">M</span><span>mock.ai</span>
        </a>
        <div>
          <p className="eyebrow">IELTS practice</p>
          <h1>Practice clearly.<br />Progress visibly.</h1>
          <p>Realistic Listening and Reading mock exams, with results that show where your score comes from.</p>
        </div>
        <small>Built for focused, repeatable practice.</small>
      </section>
      <section className="auth-panel">
        <div className="auth-card">
          <p className="eyebrow">Welcome</p>
          <h2>{mode === "login" ? "Sign in to continue" : "Create your account"}</h2>
          <p className="auth-subtitle">
            {mode === "login" ? "Your progress is waiting for you." : "Start tracking every practice result."}
          </p>
          <form onSubmit={submit}>
            {mode === "register" && (
              <label>Full name<input autoComplete="name" value={displayName} onChange={(event) => setDisplayName(event.target.value)} required minLength={2} maxLength={100} /></label>
            )}
            <label>Email<input type="email" autoComplete="email" value={email} onChange={(event) => setEmail(event.target.value)} required /></label>
            <label>Password<input type="password" autoComplete={mode === "login" ? "current-password" : "new-password"} value={password} onChange={(event) => setPassword(event.target.value)} required minLength={10} maxLength={128} /></label>
            {error && <div className="form-error" role="alert">{error}</div>}
            <button className="primary-button" disabled={submitting}>
              {submitting ? "Please wait…" : mode === "login" ? "Sign in" : "Create account"}
            </button>
          </form>
          <div className="auth-divider"><span>or</span></div>
          <a className="google-button" href="/api/v1/auth/google/start">
            <GoogleIcon /> Continue with Google
          </a>
          <button className="auth-switch" onClick={switchMode}>
            {mode === "login" ? "New here? Create an account" : "Already registered? Sign in"}
          </button>
        </div>
      </section>
    </main>
  );
}

function GoogleIcon() {
  return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M21.6 12.2c0-.7-.1-1.5-.2-2.2H12v4h5.4a4.6 4.6 0 0 1-2 3v2.6h3.3c1.9-1.8 2.9-4.4 2.9-7.4Z"/><path d="M12 22c2.7 0 5-.9 6.7-2.4L15.4 17c-.9.6-2.1 1-3.4 1a5.9 5.9 0 0 1-5.6-4.1H3v2.7A10 10 0 0 0 12 22Z"/><path d="M6.4 13.9A6 6 0 0 1 6.1 12c0-.7.1-1.3.3-1.9V7.4H3A10 10 0 0 0 2 12c0 1.6.4 3.2 1 4.6l3.4-2.7Z"/><path d="M12 6c1.5 0 2.8.5 3.9 1.5l2.9-2.9A9.8 9.8 0 0 0 12 2a10 10 0 0 0-9 5.4l3.4 2.7A5.9 5.9 0 0 1 12 6Z"/></svg>;
}

