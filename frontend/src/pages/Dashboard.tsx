import { Link } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { Plus, ChevronRight, Layers, Trash2, Edit2, Sparkles, Loader2, Check } from "lucide-react"
import { Button } from "../components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "../components/ui/card"
import { getPlan, createSegment, deleteSegment, suggestSections } from "../api"
import { useState } from "react"
import type { SectionSuggestion } from "../lib/types"

export function Dashboard() {
  const { data: plan, isLoading, error, refetch } = useQuery({ queryKey: ["plan"], queryFn: getPlan })
  const [isAdding, setIsAdding] = useState(false)
  const [newTitle, setNewTitle] = useState("")
  const [newDesc, setNewDesc] = useState("")
  const [newOrder, setNewOrder] = useState(0)

  // AI suggestions state
  const [suggestions, setSuggestions] = useState<SectionSuggestion[]>([])
  const [customPrompt, setCustomPrompt] = useState("")
  const [isSuggesting, setIsSuggesting] = useState(false)
  const [suggestError, setSuggestError] = useState("")
  const [acceptedTitles, setAcceptedTitles] = useState<Set<string>>(new Set())

  const handleAddSegment = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await createSegment({ title: newTitle, description: newDesc, order_index: newOrder })
      setIsAdding(false)
      setNewTitle("")
      setNewDesc("")
      setNewOrder(0)
      refetch()
    } catch (err) {
      console.error("Failed to create segment", err)
    }
  }

  const handleDelete = async (id: number) => {
    if (confirm("Are you sure you want to delete this segment and all its quizzes?")) {
      try {
        await deleteSegment(id)
        refetch()
      } catch (err) {
        console.error("Failed to delete segment", err)
      }
    }
  }

  const handleSuggest = async () => {
    setIsSuggesting(true)
    setSuggestError("")
    setSuggestions([])
    setAcceptedTitles(new Set())
    try {
      const data = await suggestSections(customPrompt)
      setSuggestions(data)
    } catch (err) {
      setSuggestError("Failed to get AI suggestions. Is the LLM running?")
      console.error(err)
    } finally {
      setIsSuggesting(false)
    }
  }

  const handleAcceptSuggestion = async (suggestion: SectionSuggestion) => {
    const nextOrder = (plan?.length || 0) + acceptedTitles.size + 1
    try {
      await createSegment({ title: suggestion.title, description: suggestion.description, order_index: nextOrder })
      setAcceptedTitles((prev) => new Set(prev).add(suggestion.title))
      refetch()
    } catch (err) {
      console.error("Failed to create segment from suggestion", err)
    }
  }

  if (isLoading) return <div className="flex items-center justify-center h-64 text-slate-500">Loading plan...</div>
  if (error) return <div className="text-red-500 bg-red-50 p-4 rounded-md border border-red-200">Failed to load learning plan</div>

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center bg-white p-6 rounded-lg shadow-sm border border-slate-200">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-slate-900">Learning Plan</h1>
          <p className="text-slate-500 mt-1">Manage the overall curriculum segments.</p>
        </div>
        <div className="flex gap-2">
          <div className="flex bg-slate-100 p-1 rounded-md border border-slate-200">
            <input
              type="text"
              value={customPrompt}
              onChange={(e) => setCustomPrompt(e.target.value)}
              placeholder="e.g. Travel tips..."
              className="bg-transparent border-none focus:ring-0 text-sm px-2 w-48"
            />
            <Button
              onClick={handleSuggest}
              disabled={isSuggesting}
              variant="ghost"
              size="sm"
              className="gap-2 text-purple-700 hover:bg-purple-100"
            >
              {isSuggesting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
              {isSuggesting ? "Thinking..." : "AI Suggest"}
            </Button>
          </div>
          <Button onClick={() => setIsAdding(!isAdding)} className="gap-2">
            {isAdding ? "Cancel" : <><Plus className="h-4 w-4" /> Add Segment</>}
          </Button>
        </div>
      </div>

      {suggestError && (
        <div className="text-red-600 bg-red-50 border border-red-200 p-4 rounded-lg text-sm">{suggestError}</div>
      )}

      {suggestions.length > 0 && (
        <Card className="border-purple-200 shadow-md bg-gradient-to-br from-purple-50/50 to-white">
          <CardHeader className="pb-3">
            <CardTitle className="text-lg text-purple-900 flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-purple-500" />
              AI Suggestions
            </CardTitle>
            <p className="text-sm text-slate-500 mt-1">Click Accept to add a suggestion as a new segment.</p>
          </CardHeader>
          <CardContent className="space-y-3">
            {suggestions.map((s) => {
              const isAccepted = acceptedTitles.has(s.title)
              return (
                <div
                  key={s.title}
                  className={`flex items-center justify-between p-4 rounded-lg border transition-all ${
                    isAccepted
                      ? "bg-green-50 border-green-200"
                      : "bg-white border-slate-200 hover:border-purple-300 hover:shadow-sm"
                  }`}
                >
                  <div className="flex-1 min-w-0 mr-4">
                    <h4 className="font-semibold text-slate-900">{s.title}</h4>
                    <p className="text-sm text-slate-500 mt-0.5 line-clamp-2">{s.description}</p>
                  </div>
                  {isAccepted ? (
                    <span className="flex items-center gap-1.5 text-green-700 text-sm font-medium whitespace-nowrap">
                      <Check className="h-4 w-4" /> Added
                    </span>
                  ) : (
                    <Button
                      size="sm"
                      onClick={() => handleAcceptSuggestion(s)}
                      className="whitespace-nowrap bg-purple-600 hover:bg-purple-700"
                    >
                      Accept
                    </Button>
                  )}
                </div>
              )
            })}
            <div className="flex justify-end pt-2">
              <Button variant="ghost" size="sm" onClick={() => setSuggestions([])} className="text-slate-500">
                Dismiss
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {isAdding && (
        <Card className="border-blue-100 shadow-md">
          <CardHeader className="bg-blue-50/50 pb-4">
            <CardTitle className="text-lg text-blue-900">Add New Segment</CardTitle>
          </CardHeader>
          <CardContent className="pt-4">
            <form onSubmit={handleAddSegment} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">Title</label>
                  <input
                    type="text"
                    required
                    value={newTitle}
                    onChange={(e) => setNewTitle(e.target.value)}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="e.g., Basics"
                  />
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-slate-700">Order Index</label>
                  <input
                    type="number"
                    required
                    value={newOrder}
                    onChange={(e) => setNewOrder(parseInt(e.target.value))}
                    className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700">Description</label>
                <textarea
                  required
                  value={newDesc}
                  onChange={(e) => setNewDesc(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 h-24"
                  placeholder="Describe what will be learned in this segment..."
                />
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <Button type="button" variant="outline" onClick={() => setIsAdding(false)}>Cancel</Button>
                <Button type="submit">Save Segment</Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4">
        {plan?.map((item: any) => (
          <Card key={item.segment.id} className="group hover:shadow-md transition-shadow border-slate-200">
            <CardContent className="p-0 flex items-stretch">
              <div className="bg-slate-50 w-16 flex items-center justify-center border-r border-slate-100 rounded-l-lg">
                <div className="h-8 w-8 rounded-full bg-blue-100 text-blue-700 font-bold flex items-center justify-center">
                  {item.segment?.order_index}
                </div>
              </div>
              <div className="p-6 flex-1 flex items-center justify-between">
                <div>
                  <h3 className="text-xl font-semibold text-slate-900 group-hover:text-blue-600 transition-colors">
                    {item.segment?.title}
                  </h3>
                  <p className="text-slate-500 text-sm mt-1 line-clamp-2 max-w-2xl">{item.segment?.description}</p>

                  <div className="flex items-center gap-4 mt-4 text-xs font-medium text-slate-400">
                    <span className="flex items-center gap-1.5 bg-slate-100 px-2 py-1 rounded-md text-slate-600">
                      <Layers className="h-3.5 w-3.5" />
                      {item.quizzes?.length || 0} Quizzes
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button variant="outline" size="icon" className="h-8 w-8 text-slate-400 hover:text-blue-600">
                      <Edit2 className="h-4 w-4" />
                    </Button>
                    <Button variant="outline" size="icon" onClick={() => handleDelete(item.segment?.id)} className="h-8 w-8 text-slate-400 hover:text-red-600 hover:bg-red-50 hover:border-red-200">
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                  <Link to={`/segments/${item.segment?.id}`}>
                    <Button variant="ghost" className="gap-1 text-blue-600 hover:bg-blue-50">
                      Manage <ChevronRight className="h-4 w-4" />
                    </Button>
                  </Link>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
        {plan?.length === 0 && !isAdding && (
          <div className="text-center py-12 bg-white rounded-lg border border-dashed border-slate-300">
            <Layers className="mx-auto h-12 w-12 text-slate-300" />
            <h3 className="mt-2 text-sm font-medium text-slate-900">No segments</h3>
            <p className="mt-1 text-sm text-slate-500">Get started by creating a new segment.</p>
            <div className="mt-6">
              <Button onClick={() => setIsAdding(true)}><Plus className="h-4 w-4 mr-2"/> Add Segment</Button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}