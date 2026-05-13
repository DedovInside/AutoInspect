import Header from "../Header/Header";
import { Outlet } from "react-router-dom";
import AnalysisNotifications from "../AnalysisNotifications/AnalysisNotifications";
import "./Layout.css";

function Layout() {
  return (
    <div className="app-shell">
      <Header />
      <AnalysisNotifications />
      <main className="app-main">
        <div className="container">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

export default Layout;
