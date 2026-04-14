import Header from "../Header/Header";
import { Outlet } from "react-router-dom";

function Layout() {
  return (
    <div>
      <Header />

      <main
        style={{
          width: "100%",
          padding: "40px 60px",
        }}
      >
        <Outlet />
      </main>
    </div>
  );
}

export default Layout;