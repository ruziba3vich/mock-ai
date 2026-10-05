import { apiRequest } from "./auth";

export type ExamSummary = {
  id: string; slug: string; title: string; description: string;
  skill: "listening" | "reading"; duration_seconds: number;
  question_count: number; published_at: string;
};
export type Option = { id: string; position: number; label: string; value: string };
export type Question = { id: string; position: number; type: string; prompt: string; points: number; config: Record<string, unknown>; options: Option[] };
export type Section = { id: string; position: number; title: string; instructions: string; audio_url?: string; passage?: string; questions: Question[] };
export type Exam = ExamSummary & { version_id: string; version: number; instructions: string; sections: Section[] };
export type SavedAnswer = { question_id: string; values: string[]; revision: number; saved_at: string; is_correct?: boolean; awarded_points?: number };
export type Attempt = {
  id: string; exam_version_id: string; status: string; started_at: string; deadline_at: string;
  submitted_at?: string; raw_score?: number; band_score?: number; question_count: number; answers: SavedAnswer[];
};
export type HistoryItem = Attempt & { title: string; skill: string };
export type SkillProgress = { skill: string; completed_attempts: number; average_band: number; best_band: number; latest_band: number; correct_answers: number; total_questions: number };

export const examAPI = {
  list: async () => (await apiRequest<{ exams: ExamSummary[] }>("/api/v1/exams")).exams,
  get: async (id: string) => (await apiRequest<{ exam: Exam }>(`/api/v1/exams/${id}`)).exam,
  start: async (id: string) => (await apiRequest<{ attempt: Attempt }>(`/api/v1/exams/${id}/attempts`, { method: "POST" })).attempt,
  save: async (attemptID: string, questionID: string, values: string[]) =>
    (await apiRequest<{ answer: SavedAnswer }>(`/api/v1/attempts/${attemptID}/answers/${questionID}`, { method: "PUT", body: JSON.stringify({ values }) })).answer,
  submit: async (attemptID: string) => (await apiRequest<{ attempt: Attempt }>(`/api/v1/attempts/${attemptID}/submit`, { method: "POST" })).attempt,
  history: async () => (await apiRequest<{ attempts: HistoryItem[] }>("/api/v1/me/attempts?limit=5")).attempts,
  progress: async () => (await apiRequest<{ skills: SkillProgress[] }>("/api/v1/me/progress")).skills
};
