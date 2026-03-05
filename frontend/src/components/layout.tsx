import { Link, Outlet } from "react-router-dom"
import { BookOpen } from "lucide-react"

export function Layout() {
  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <header className="bg-white border-b border-slate-200 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <Link to="/" className="flex items-center space-x-2 text-blue-600 hover:text-blue-700 transition-colors">
            <BookOpen className="h-6 w-6" />
            <span className="font-bold text-xl tracking-tight text-slate-900">QuizBot Admin</span>
          </Link>
          <nav className="flex space-x-4">
            <Link to="/" className="text-slate-600 hover:text-slate-900 font-medium px-3 py-2 rounded-md transition-colors hover:bg-slate-100">
              Dashboard
            </Link>
          </nav>
        </div>
      </header>

      <main className="flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <Outlet />
      </main>

      <footer className="bg-white border-t border-slate-200 mt-auto">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-center text-sm text-slate-500">
          Learning Curriculum Manager
        </div>
      </footer>
    </div>
  )
}