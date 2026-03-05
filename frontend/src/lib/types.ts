export interface Segment {
  id: number;
  title: string;
  description: string;
  order_index: number;
  created_at: string;
}

export interface Quiz {
  id: number;
  segment_id: number;
  title: string;
  description: string;
  order_index: number;
  created_at: string;
}

export interface Question {
  id: number;
  quiz_id: number;
  text: string;
  options: string[];
  correct_answer: string;
  audio_file_id?: string;
  is_active: boolean;
  created_at: string;
}

export interface PlanItem {
  segment: Segment;
  quizzes: {
    quiz: Quiz;
    questions: Question[];
  }[];
}