import { useEffect, useMemo, useRef, useState } from "react";
import { Attempt, Exam, examAPI, Question } from "./exams";

type Props = { exam: Exam; initialAttempt: Attempt; onExit: () => void };

export default function ExamRunner({ exam, initialAttempt, onExit }: Props) {
  const [attempt, setAttempt] = useState(initialAttempt);
  const [answers, setAnswers] = useState<Record<string, string[]>>(() => Object.fromEntries(initialAttempt.answers.map((a) => [a.question_id, a.values])));
  const [saving, setSaving] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [now, setNow] = useState(Date.now());
  const expirySubmitted = useRef(false);
  const questions = useMemo(() => exam.sections.flatMap((section) => section.questions), [exam]);

  useEffect(() => {
    if (attempt.status !== "in_progress") return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [attempt.status]);

  const remaining = Math.max(0, Math.floor((new Date(attempt.deadline_at).getTime() - now) / 1000));
  const clock = `${String(Math.floor(remaining / 60)).padStart(2, "0")}:${String(remaining % 60).padStart(2, "0")}`;

  useEffect(() => {
    if (remaining === 0 && attempt.status === "in_progress" && !expirySubmitted.current) {
      expirySubmitted.current = true;
      void submit();
    }
  });

  async function save(questionID: string, values = answers[questionID] || []) {
    if (!values.length || attempt.status !== "in_progress") return;
    setSaving(questionID); setError("");
    try { await examAPI.save(attempt.id, questionID, values); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "Answer could not be saved."); }
    finally { setSaving(null); }
  }

  async function submit() {
    setSubmitting(true); setError("");
    try { setAttempt(await examAPI.submit(attempt.id)); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "Attempt could not be submitted."); }
    finally { setSubmitting(false); }
  }

  if (attempt.status === "scored") {
    return <main className="result-page"><div className="result-card"><p className="eyebrow">Exam complete</p><h1>Band {attempt.band_score?.toFixed(1)}</h1><p>You answered <strong>{attempt.raw_score}</strong> of <strong>{attempt.question_count}</strong> questions correctly.</p><button className="primary-button" onClick={onExit}>Back to dashboard</button></div></main>;
  }

  return <main className="runner">
    <header className="runner-header"><button onClick={onExit}>← Exit</button><div><strong>{exam.title}</strong><span>{exam.skill}</span></div><time className={remaining < 300 ? "urgent" : ""}>{clock}</time></header>
    <div className="runner-body">
      <aside><p className="eyebrow">Questions</p><div className="question-map">{questions.map((q, i) => <a key={q.id} className={answers[q.id]?.length ? "answered" : ""} href={`#question-${q.id}`}>{i + 1}</a>)}</div><small>{Object.values(answers).filter((v) => v.length).length} of {questions.length} answered</small></aside>
      <div className="exam-content">
        {exam.sections.map((section) => <section className="runner-section" key={section.id}>
          <div className="material"><p className="eyebrow">Section {section.position}</p><h2>{section.title}</h2>{section.instructions && <p>{section.instructions}</p>}{section.audio_url && <audio controls src={section.audio_url} />}{section.passage && <article>{section.passage}</article>}</div>
          <div className="questions">{section.questions.map((question) => <QuestionInput key={question.id} question={question} value={answers[question.id] || []} saving={saving === question.id} onChange={(values, immediate) => { setAnswers((current) => ({ ...current, [question.id]: values })); if (immediate) void save(question.id, values); }} onBlur={() => void save(question.id)} />)}</div>
        </section>)}
        {error && <div className="form-error" role="alert">{error}</div>}
        <button className="submit-exam" disabled={submitting} onClick={() => void submit()}>{submitting ? "Scoring…" : "Submit exam"}</button>
      </div>
    </div>
  </main>;
}

function QuestionInput({ question, value, saving, onChange, onBlur }: { question: Question; value: string[]; saving: boolean; onChange: (value: string[], immediate: boolean) => void; onBlur: () => void }) {
  const multiple = question.type === "multiple_select";
  return <article className="question" id={`question-${question.id}`}><div className="question-number">{question.position}</div><div><h3>{question.prompt}</h3>
    {question.options.length > 0 ? <div className="options">{question.options.map((option) => { const selected = value.includes(option.value); return <label className={selected ? "selected" : ""} key={option.id}><input type={multiple ? "checkbox" : "radio"} name={question.id} checked={selected} onChange={() => { const next = multiple ? (selected ? value.filter((v) => v !== option.value) : [...value, option.value]) : [option.value]; onChange(next, next.length > 0); }} /><span>{option.label}</span>{option.value}</label>; })}</div> : <input className="answer-input" value={value[0] || ""} onChange={(event) => onChange([event.target.value], false)} onBlur={onBlur} placeholder="Type your answer" />}
    <small className="save-state">{saving ? "Saving…" : value.length && value[0] ? "Saved automatically" : ""}</small>
  </div></article>;
}
