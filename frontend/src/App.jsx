import { BrowserRouter, Routes, Route } from 'react-router-dom';

import HomePage from './pages/HomePage/HomePage';
import LoginPage from './pages/LoginPage/LoginPage';
import RegistrationPage from './pages/RegistrationPage/RegistrationPage';
import UploadPage from './pages/UploadPage/UploadPage';
import ResultPage from './pages/ResultPage/ResultPage';
import HistoryPage from './pages/HistoryPage/HistoryPage';
import AdminPage from './pages/AdminPage/AdminPage';

function Layout({ children }) {
  return (
    <div>
      <header>
        <h1>AutoInspect</h1>
      </header>
      <main>{children}</main>
    </div>
  );
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout><HomePage /></Layout>} />
        <Route path="/login" element={<Layout><LoginPage /></Layout>} />
        <Route path="/registration" element={<Layout><RegistrationPage /></Layout>} />
        <Route path="/upload" element={<Layout><UploadPage /></Layout>} />
        <Route path="/result" element={<Layout><ResultPage /></Layout>} />
        <Route path="/history" element={<Layout><HistoryPage /></Layout>} />
        <Route path="/admin" element={<Layout><AdminPage /></Layout>} />
        <Route path="*" element={<div>404 Not Found</div>} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
