import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import AuthPage from './pages/AuthPage/AuthPage';
import AuthCallbackPage from './pages/AuthCallbackPage/AuthCallbackPage';
import HomePage from './pages/HomePage/HomePage';
import UploadPage from './pages/UploadPage/UploadPage';
import ResultPage from './pages/ResultPage/ResultPage';
import HistoryPage from './pages/HistoryPage/HistoryPage';
import AdminPage from './pages/AdminPage/AdminPage';
import ServiceProfilePage from './pages/ServiceProfilePage/ServiceProfilePage';
import RepairRequestsPage from './pages/RepairRequestsPage/RepairRequestsPage';
import ProtectedRoute from './components/ProtectedRoute/ProtectedRoute';
import RoleBasedRoute from './components/RoleBasedRoute/RoleBasedRoute';
import Layout from './components/Layout/Layout';
import { useAuth } from './auth/AuthContext';

function App() {
  const { isAuthenticated } = useAuth();
  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/"
          element={
            isAuthenticated ? <Navigate to="/home" /> : <Navigate to="/auth" />
          }
        />

        <Route path="/auth" element={<AuthPage />} />
        <Route path="/auth/callback" element={<AuthCallbackPage />} />

        <Route element={<Layout />}>
          <Route
            path="/home"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["USER", "SERVICE", "ADMIN"]}>
                  <HomePage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />

          <Route
            path="/upload"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["USER", "SERVICE", "ADMIN"]}>
                  <UploadPage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />

          <Route
            path="/result/:id"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["USER", "SERVICE", "ADMIN"]}>
                  <ResultPage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />

          <Route
            path="/history"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["USER", "SERVICE", "ADMIN"]}>
                  <HistoryPage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />

          <Route
            path="/repair-requests"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["USER", "SERVICE", "ADMIN"]}>
                  <RepairRequestsPage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />

          <Route
            path="/service-profile"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["SERVICE"]}>
                  <ServiceProfilePage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />

          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <RoleBasedRoute allowedRoles={["ADMIN"]}>
                  <AdminPage />
                </RoleBasedRoute>
              </ProtectedRoute>
            }
          />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
