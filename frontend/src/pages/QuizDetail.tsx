import { useParams, Link } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { Plus, ArrowLeft, HelpCircle, Trash2, Mic, ChevronRight } from "lucide-react"
import { Button } from "../components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "../components/ui/card"
import { getQuiz, getQuizQuestions, createQuestion, deleteQuestion } from "../api"
import { useState } from "react"

export function QuizDetail() {
  const { id } = useParams()
  const quizId = parseInt(id || "0")

  const { data: quiz, isLoading: isLoadingQuiz } = useQuery({ queryKey: ["quiz", quizId], queryFn: () => getQuiz(quizId) })
  const { data: questions, isLoading: isLoadingQuestions, refetch } = useQuery({ queryKey: ["quizQuestions", quizId], queryFn: () => getQuizQuestions(quizId) })

  const [isAdding, setIsAdding] = useState(false)
  const [newText, setNewText] = useState("")
  const [options, setOptions] = useState(["", "", "", ""])
  const [correctAnswer, setCorrectAnswer] = useState(0)

  const handleAddQuestion = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const validOptions = options.filter(o => o.trim() !== "")
      if (validOptions.length < 2) {
        alert("Please provide at least 2 options")
        return
      }

      await createQuestion(quizId, {
        text: newText,
        options: validOptions,
        correct_answer: options[correctAnswer],
        quiz_id: quizId,
        is_active: true
      })

      setIsAdding(false)
      setNewText("")
      setOptions(["", "", "", ""])
      setCorrectAnswer(0)
      refetch()
    } catch (err) {
      console.error("Failed to create question", err)
    }
  }

  const handleDelete = async (questionId: number) => {
    if (confirm("Are you sure you want to delete this question?")) {
      try {
        await deleteQuestion(questionId)
        refetch()
      } catch (err) {
        console.error("Failed to delete question", err)
      }
    }
  }

  if (isLoadingQuiz || isLoadingQuestions) return <div className="flex items-center justify-center h-64 text-slate-500">Loading quiz details...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 text-sm text-slate-500 mb-4">
        <Link to={`/segments/${quiz?.segment_id}`} className="hover:text-blue-600 flex items-center gap-1 transition-colors">
          <ArrowLeft className="h-4 w-4" /> Back to Segment
        </Link>
        <ChevronRight className="h-4 w-4" />
        <span className="font-medium text-slate-900">{quiz?.title}</span>
      </div>

      <div className="flex justify-between items-center bg-white p-6 rounded-lg shadow-sm border border-slate-200">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900">{quiz?.title}</h1>
          <p className="text-slate-500 mt-1 max-w-3xl">{quiz?.description}</p>
        </div>
        <Button onClick={() => setIsAdding(!isAdding)} className="gap-2 bg-emerald-600 hover:bg-emerald-700">
          {isAdding ? "Cancel" : <><Plus className="h-4 w-4" /> Add Question</>}
        </Button>
      </div>

      {isAdding && (
        <Card className="border-emerald-100 shadow-md">
          <CardHeader className="bg-emerald-50/50 pb-4">
            <CardTitle className="text-lg text-emerald-900">Add New Question</CardTitle>
          </CardHeader>
          <CardContent className="pt-4">
            <form onSubmit={handleAddQuestion} className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">Question Text</label>
                <textarea
                  required
                  value={newText}
                  onChange={(e) => setNewText(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500 h-24"
                  placeholder="e.g., How do you say 'Hello' in Spanish?"
                />
              </div>

              <div className="space-y-3">
                <label className="text-sm font-medium text-slate-700">Options</label>
                {options.map((opt, idx) => (
                  <div key={idx} className="flex items-center gap-3">
                    <input
                      type="radio"
                      name="correctAnswer"
                      checked={correctAnswer === idx}
                      onChange={() => setCorrectAnswer(idx)}
                      className="h-4 w-4 text-emerald-600 focus:ring-emerald-500"
                    />
                    <input
                      type="text"
                      value={opt}
                      onChange={(e) => {
                        const newOpts = [...options]
                        newOpts[idx] = e.target.value
                        setOptions(newOpts)
                      }}
                      className="flex-1 px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500"
                      placeholder={`Option ${idx + 1}`}
                      required={idx < 2} // Require at least first two
                    />
                    {correctAnswer === idx && <span className="text-xs text-emerald-600 font-medium">Correct Answer</span>}
                  </div>
                ))}
              </div>

              <div className="flex justify-end gap-2 pt-2 border-t border-slate-100 mt-4">
                <Button type="button" variant="outline" onClick={() => setIsAdding(false)}>Cancel</Button>
                <Button type="submit" className="bg-emerald-600 hover:bg-emerald-700">Save Question</Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4">
        {questions?.map((q: any, i: number) => (
          <Card key={q.id} className="group hover:shadow-md transition-shadow border-slate-200">
             <CardContent className="p-6">
              <div className="flex justify-between items-start mb-4">
                <div className="flex items-start gap-3">
                   <div className="bg-emerald-100 text-emerald-700 rounded-full h-8 w-8 flex items-center justify-center font-bold text-sm shrink-0 mt-0.5">
                     Q{i+1}
                   </div>
                   <div>
                     <h3 className="text-lg font-medium text-slate-900">{q.text}</h3>
                     {q.audio_file_id && (
                       <span className="inline-flex items-center gap-1 text-xs text-blue-600 bg-blue-50 px-2 py-1 rounded-md mt-2 border border-blue-100">
                         <Mic className="h-3 w-3" /> Audio attached
                       </span>
                     )}
                   </div>
                </div>
                <Button variant="ghost" size="icon" onClick={() => handleDelete(q.id)} className="h-8 w-8 text-slate-400 hover:text-red-600 hover:bg-red-50 hover:border-red-200 shrink-0">
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 pl-11">
                {q.options.map((opt: string, idx: number) => {
                  const isCorrect = opt === q.correct_answer
                  return (
                    <div
                      key={idx}
                      className={`p-3 rounded-md border text-sm ${isCorrect ? 'bg-emerald-50 border-emerald-200 text-emerald-800 font-medium' : 'bg-slate-50 border-slate-200 text-slate-700'}`}
                    >
                      {opt} {isCorrect && " ✓"}
                    </div>
                  )
                })}
              </div>
            </CardContent>
          </Card>
        ))}
        {questions?.length === 0 && !isAdding && (
          <div className="text-center py-12 bg-white rounded-lg border border-dashed border-slate-300">
            <HelpCircle className="mx-auto h-12 w-12 text-slate-300" />
            <h3 className="mt-2 text-sm font-medium text-slate-900">No questions yet</h3>
            <p className="mt-1 text-sm text-slate-500">The LLM hasn't generated any questions, or you haven't added any.</p>
            <div className="mt-6">
              <Button onClick={() => setIsAdding(true)} className="bg-emerald-600 hover:bg-emerald-700"><Plus className="h-4 w-4 mr-2"/> Add First Question</Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
