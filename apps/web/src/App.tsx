type Exam = {
  skill: "Listening" | "Reading";
  title: string;
  meta: string;
  tone: "violet" | "green";
};

const exams: Exam[] = [
  { skill: "Listening", title: "Cambridge practice 01", meta: "40 questions · 30 min", tone: "violet" },
  { skill: "Reading", title: "Academic practice 01", meta: "40 questions · 60 min", tone: "green" }
];

function App() {
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
          <div className="avatar">AK</div>
          <div><strong>Aziz Karimov</strong><span>Student</span></div>
        </div>
      </aside>

      <main>
        <header className="mobile-header">
          <a className="brand" href="#"><span className="brand-mark">M</span><span>mock.ai</span></a>
          <button aria-label="Open profile" className="avatar small">AK</button>
        </header>

        <div className="content">
          <section className="hero" id="dashboard">
            <div>
              <p className="eyebrow">Monday, 5 October</p>
              <h1>Keep your English<br />moving forward.</h1>
              <p className="hero-copy">Practice under real exam conditions and see exactly where to improve.</p>
            </div>
            <div className="band-card">
              <span>Current average</span>
              <strong>6.5</strong>
              <small>+0.5 in the last month</small>
            </div>
          </section>

          <section className="section" id="exams">
            <div className="section-heading">
              <div><p className="eyebrow">Practice</p><h2>Choose your next exam</h2></div>
              <a href="#all-exams">View all <span>→</span></a>
            </div>
            <div className="exam-grid">
              {exams.map((exam) => (
                <article className={`exam-card ${exam.tone}`} key={exam.skill}>
                  <div className="exam-icon">{exam.skill === "Listening" ? <HeadphonesIcon /> : <BookIcon />}</div>
                  <span className="skill">{exam.skill}</span>
                  <h3>{exam.title}</h3>
                  <p>{exam.meta}</p>
                  <button>Start exam <span>→</span></button>
                </article>
              ))}
            </div>
          </section>

          <section className="section" id="progress">
            <div className="section-heading">
              <div><p className="eyebrow">Your progress</p><h2>Recent performance</h2></div>
              <a href="#history">Full history <span>→</span></a>
            </div>
            <div className="progress-card">
              <div className="score-block"><span>Reading</span><strong>7.0</strong><small>Last attempt</small></div>
              <div className="bar-area">
                <div><span>Correct answers</span><strong>32 / 40</strong></div>
                <div className="bar"><span style={{ width: "80%" }} /></div>
              </div>
              <div className="score-block"><span>Listening</span><strong>6.0</strong><small>Last attempt</small></div>
              <div className="bar-area">
                <div><span>Correct answers</span><strong>26 / 40</strong></div>
                <div className="bar"><span style={{ width: "65%" }} /></div>
              </div>
            </div>
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

function GridIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="2"/><rect x="14" y="3" width="7" height="7" rx="2"/><rect x="3" y="14" width="7" height="7" rx="2"/><rect x="14" y="14" width="7" height="7" rx="2"/></svg>; }
function BookIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H11v16H6.5A2.5 2.5 0 0 0 4 21.5v-16ZM20 5.5A2.5 2.5 0 0 0 17.5 3H13v16h4.5a2.5 2.5 0 0 1 2.5 2.5v-16Z"/></svg>; }
function ChartIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20V10m6 10V4m6 16v-7m4 7H2"/></svg>; }
function HeadphonesIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 14v-2a8 8 0 0 1 16 0v2M4 14h3v7H6a2 2 0 0 1-2-2v-5Zm16 0h-3v7h1a2 2 0 0 0 2-2v-5Z"/></svg>; }

export default App;

