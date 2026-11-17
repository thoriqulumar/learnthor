import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import { useAuth } from './context/AuthContext';
import ProtectedLayout from './components/ProtectedLayout';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import HomePage from './pages/HomePage';
import ModulePage from './pages/ModulePage';
import VocabPage from './pages/VocabPage';
import QuizPage from './pages/QuizPage';
import SectionQuizPage from './pages/SectionQuizPage';
import LearnPage from './pages/LearnPage';
import ReviewPage from './pages/ReviewPage';
import UpgradePage from './pages/UpgradePage';
import MenuPage from './pages/MenuPage';
import KanjiPage from './pages/KanjiPage';
import KanjiDetailPage from './pages/KanjiDetailPage';
import LearningLevelPage from './pages/LearningLevelPage';
import JLPTLessonPage from './pages/JLPTLessonPage';
import JLPTExamPage from './pages/JLPTExamPage';

// Redirects logged-in users away from auth pages (login / register)
function GuestRoute({ element }) {
  const { user, loading } = useAuth();
  if (loading) return null;
  return user ? <Navigate to="/" replace /> : element;
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route
            path="/login"
            element={<GuestRoute element={<LoginPage />} />}
          />
          <Route
            path="/register"
            element={<GuestRoute element={<RegisterPage />} />}
          />
          <Route element={<ProtectedLayout />}>
            <Route path="/" element={<MenuPage />} />
            <Route path="/learn" element={<HomePage />} />
            <Route path="/learn/:level" element={<LearningLevelPage />} />
            <Route path="/learn/:level/exam" element={<JLPTExamPage />} />
            <Route path="/learn/:level/lessons/:lessonId" element={<JLPTLessonPage />} />
            <Route path="/kanji" element={<KanjiPage />} />
            <Route path="/kanji/:character" element={<KanjiDetailPage />} />
            <Route path="/modules/:moduleId" element={<ModulePage />} />
            <Route path="/modules/:moduleId/vocab" element={<VocabPage />} />
            <Route path="/modules/:moduleId/quiz" element={<QuizPage />} />
            <Route path="/modules/:moduleId/learn" element={<LearnPage />} />
            <Route path="/review" element={<ReviewPage />} />
            <Route path="/upgrade" element={<UpgradePage />} />
            <Route
              path="/sections/:sectionId/quiz"
              element={<SectionQuizPage />}
            />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
