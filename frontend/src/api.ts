export const API_URL = "/api"

export async function getPlan() {
  const res = await fetch(`${API_URL}/plan`)
  if (!res.ok) throw new Error("Failed to fetch plan")
  return res.json()
}

export async function getSegments() {
  const res = await fetch(`${API_URL}/segments`)
  if (!res.ok) throw new Error("Failed to fetch segments")
  return res.json()
}

export async function getSegment(id: number) {
  const res = await fetch(`${API_URL}/segments/${id}`)
  if (!res.ok) throw new Error("Failed to fetch segment")
  return res.json()
}

export async function createSegment(data: any) {
  const res = await fetch(`${API_URL}/segments`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error("Failed to create segment")
  return res.json()
}

export async function deleteSegment(id: number) {
  const res = await fetch(`${API_URL}/segments/${id}`, { method: "DELETE" })
  if (!res.ok) throw new Error("Failed to delete segment")
  return res.json()
}

export async function getSegmentQuizzes(segmentId: number) {
  const res = await fetch(`${API_URL}/segments/${segmentId}/quizzes`)
  if (!res.ok) throw new Error("Failed to fetch quizzes")
  return res.json()
}

export async function createQuiz(segmentId: number, data: any) {
  const res = await fetch(`${API_URL}/segments/${segmentId}/quizzes`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error("Failed to create quiz")
  return res.json()
}

export async function getQuiz(id: number) {
  const res = await fetch(`${API_URL}/quizzes/${id}`)
  if (!res.ok) throw new Error("Failed to fetch quiz")
  return res.json()
}

export async function deleteQuiz(id: number) {
  const res = await fetch(`${API_URL}/quizzes/${id}`, { method: "DELETE" })
  if (!res.ok) throw new Error("Failed to delete quiz")
  return res.json()
}

export async function getQuizQuestions(quizId: number) {
  const res = await fetch(`${API_URL}/quizzes/${quizId}/questions`)
  if (!res.ok) throw new Error("Failed to fetch questions")
  return res.json()
}

export async function createQuestion(quizId: number, data: any) {
  const res = await fetch(`${API_URL}/quizzes/${quizId}/questions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error("Failed to create question")
  return res.json()
}

export async function deleteQuestion(id: number) {
  const res = await fetch(`${API_URL}/questions/${id}`, { method: "DELETE" })
  if (!res.ok) throw new Error("Failed to delete question")
  return res.json()
}