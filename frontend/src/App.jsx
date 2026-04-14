import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import AuthPage from './pages/AuthPage/AuthPage';
import AuthCallbackPage from './pages/AuthCallbackPage/AuthCallbackPage';
import HomePage from './pages/HomePage/HomePage';
import UploadPage from './pages/UploadPage/UploadPage';
import ResultPage from './pages/ResultPage/ResultPage';
import HistoryPage from './pages/HistoryPage/HistoryPage';
import AdminPage from './pages/AdminPage/AdminPage';
import ProtectedRoute from './components/ProtectedRoute/ProtectedRoute';
import Layout from './components/Layout/Layout';
import { hasAccessToken } from './services/authService';

function App() {
  const isAuth = hasAccessToken();
  return (
    <BrowserRouter>
      <Routes>
        <Route 
          path="/" 
          element={
            isAuth ? <Navigate to="/home" /> : <Navigate to="/auth" />
          } 
        />

        <Route path="/auth" element={<AuthPage />} />
        <Route path="/auth/callback" element={<AuthCallbackPage />} />

        <Route element={<Layout />}>
          <Route 
            path="/home" 
            element={
              <ProtectedRoute>
                <HomePage />
              </ProtectedRoute>
            } 
          />

          <Route 
            path="/upload" 
            element={
              <ProtectedRoute>
                <UploadPage />
              </ProtectedRoute>
            } 
          />

          <Route 
            path="/result" 
            element={
              <ProtectedRoute>
                <ResultPage />
              </ProtectedRoute>
            } 
          />

          <Route 
            path="/history" 
            element={
              <ProtectedRoute>
                <HistoryPage />
              </ProtectedRoute>
            } 
          />

          <Route 
            path="/admin" 
            element={
              <ProtectedRoute>
                <AdminPage />
              </ProtectedRoute>
            } 
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;