import { useEffect, useState } from "react";
import AuthScreen from "./AuthScreen";
import ExamRunner from "./ExamRunner";
import { APIError, authAPI, User } from "./auth";
import { Attempt, Exam, ExamSummary, examAPI, HistoryItem, SkillProgress } from "./exams";

function App() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [runner, setRunner] = useState<{ exam: Exam; attempt: Attempt } | null>(null);

  useEffect(() => {
    let active = true;
    async function restoreSession() {
      try {
        const current = await authAPI.me();
        if (active) setUser(current);
      } catch (error) {
        const initData = window.Telegram?.WebApp?.initData;
        if (initData && error instanceof APIError && error.status === 401) {
          try {
            const telegramUser = await authAPI.telegram(initData);
            if (active) setUser(telegramUser);
          } catch {
            // The regular login screen remains available if Telegram auth fails.
          }
        }
      } finally {
        if (active) setLoading(false);
      }
    }
    window.Telegram?.WebApp?.ready();
    window.Telegram?.WebApp?.expand();
    void restoreSession();
    return () => { active = false; };
  }, []);

  if (loading) return <LoadingScreen />;
  if (!user) return <AuthScreen onAuthenticated={setUser} />;
  if (runner) return <ExamRunner exam={runner.exam} initialAttempt={runner.attempt} onExit={() => setRunner(null)} />;
  return <Dashboard user={user} onOpenExam={async (examID) => {
    const [exam, attempt] = await Promise.all([examAPI.get(examID), examAPI.start(examID)]);
    setRunner({ exam, attempt });
  }} onLogout={async () => { await authAPI.logout(); setUser(null); }} />;
}

function Dashboard({ user, onLogout, onOpenExam }: { user: User; onLogout: () => Promise<void>; onOpenExam: (examID: string) => Promise<void> }) {
  const initials = user.display_name.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
  const [exams, setExams] = useState<ExamSummary[]>([]);
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [progress, setProgress] = useState<SkillProgress[]>([]);
  const [dataError, setDataError] = useState("");
  const [opening, setOpening] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    Promise.all([examAPI.list(), examAPI.history(), examAPI.progress()]).then(([examList, attempts, skills]) => {
      if (active) { setExams(examList); setHistory(attempts); setProgress(skills); }
    }).catch((error) => { if (active) setDataError(error instanceof Error ? error.message : "Dashboard could not be loaded."); });
    return () => { active = false; };
  }, []);

  const completedAttempts = progress.reduce((sum, item) => sum + item.completed_attempts, 0);
  const average = completedAttempts ? progress.reduce((sum, item) => sum + item.average_band * item.completed_attempts, 0) / completedAttempts : null;
  const progressBySkill = (skill: string) => progress.find((item) => item.skill === skill);
  const formattedDate = new Intl.DateTimeFormat("en", { weekday: "long", day: "numeric", month: "long" }).format(new Date());

  async function openExam(id: string) {
    setOpening(id); setDataError("");
    try { await onOpenExam(id); }
    catch (error) { setDataError(error instanceof Error ? error.message : "Exam could not be opened."); setOpening(null); }
  }
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <a className="brand" href="#" aria-label="Mock AI home">
          <span className="brand-mark">M</span>
          <span>mock.ai</span>
        </a>
        <nav aria-label="Main navigation">
          <a className="nav-item active" href="#dashboard"><GridIcon /> Dashboard</a>
          <a className="nav-item" href="#exams"><BookIcon /> Mock exams</a>
          <a className="nav-item" href="#progress"><ChartIcon /> Progress</a>
        </nav>
        <div className="sidebar-footer">
          {user.avatar_url ? <img className="avatar" src={user.avatar_url} alt="" /> : <div className="avatar">{initials}</div>}
          <div><strong>{user.display_name}</strong><span>{user.roles.includes("admin") ? "Admin" : "Student"}</span></div>
          <button className="logout-button" aria-label="Sign out" title="Sign out" onClick={() => void onLogout()}>↗</button>
        </div>
      </aside>

      <main>
        <header className="mobile-header">
          <a className="brand" href="#"><span className="brand-mark">M</span><span>mock.ai</span></a>
          <button aria-label="Sign out" className="avatar small" onClick={() => void onLogout()}>{initials}</button>
        </header>

        <div className="content">
          <section className="hero" id="dashboard">
            <div>
              <p className="eyebrow">{formattedDate}</p>
              <h1>Keep your English<br />moving forward.</h1>
              <p className="hero-copy">Practice under real exam conditions and see exactly where to improve.</p>
            </div>
            <div className="band-card">
              <span>Current average</span>
              <strong>{average === null ? "—" : average.toFixed(1)}</strong>
              <small>{completedAttempts ? `${completedAttempts} completed attempt${completedAttempts === 1 ? "" : "s"}` : "Complete an exam to establish your score"}</small>
            </div>
          </section>

          <section className="section" id="exams">
            <div className="section-heading">
              <div><p className="eyebrow">Practice</p><h2>Choose your next exam</h2></div>
              <a href="#all-exams">View all <span>→</span></a>
            </div>
            <div className="exam-grid">
              {exams.map((exam, index) => (
                <article className={`exam-card ${index % 2 ? "green" : "violet"}`} key={exam.id}>
                  <div className="exam-icon">{exam.skill === "listening" ? <HeadphonesIcon /> : <BookIcon />}</div>
                  <span className="skill">{exam.skill}</span>
                  <h3>{exam.title}</h3>
                  <p>{exam.question_count} questions · {Math.round(exam.duration_seconds / 60)} min</p>
                  <button disabled={opening === exam.id} onClick={() => void openExam(exam.id)}>{opening === exam.id ? "Opening…" : "Start exam"} <span>→</span></button>
                </article>
              ))}
              {!exams.length && <div className="empty-state">No exams are published yet. New Listening and Reading tests will appear here.</div>}
            </div>
            {dataError && <div className="form-error dashboard-error">{dataError}</div>}
          </section>

          <section className="section" id="progress">
            <div className="section-heading">
              <div><p className="eyebrow">Your progress</p><h2>Recent performance</h2></div>
              <a href="#history">Full history <span>→</span></a>
            </div>
            <div className="progress-card">{["reading", "listening"].map((skill) => { const item = progressBySkill(skill); const percent = item?.total_questions ? Math.round((item.correct_answers / item.total_questions) * 100) : 0; return <div className="progress-pair" key={skill}><div className="score-block"><span>{skill}</span><strong>{item?.latest_band.toFixed(1) || "—"}</strong><small>{item ? `Best ${item.best_band.toFixed(1)}` : "No attempts"}</small></div><div className="bar-area"><div><span>Overall accuracy</span><strong>{item ? `${Math.round(item.correct_answers)} / ${item.total_questions}` : "—"}</strong></div><div className="bar"><span style={{ width: `${percent}%` }} /></div></div></div>; })}</div>
          </section>
        </div>

        <nav className="bottom-nav" aria-label="Mobile navigation">
          <a className="active" href="#dashboard"><GridIcon /><span>Home</span></a>
          <a href="#exams"><BookIcon /><span>Exams</span></a>
          <a href="#progress"><ChartIcon /><span>Progress</span></a>
        </nav>
      </main>
    </div>
  );
}

function LoadingScreen() {
  return <div className="loading-screen"><span className="brand-mark">M</span><span>Preparing your practice space…</span></div>;
}

function GridIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/></svg>; }
function BookIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H11v16H6.5A2.5 2.5 0 0 0 4 21.5v-16ZM20 5.5A2.5 2.5 0 0 0 17.5 3H13v16h4.5a2.5 2.5 0 0 1 2.5 2.5v-16Z"/></svg>; }
function ChartIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20V10m6 10V4m6 16v-7m4 7H2"/></svg>; }
function HeadphonesIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 14v-2a8 8 0 0 1 16 0v2M4 14h3v7H6a2 2 0 0 1-2-2v-5Zm16 0h-3v7h1a2 2 0 0 0 2-2v-5Z"/></svg>; }

export default App;
