import "./index.css";

import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

import { AdminLayout } from "./components/layout/AdminLayout";
import { GuestLayout } from "./components/layout/GuestLayout";
import { SignedInLayout } from "./components/layout/SignedInLayout";
import { BasicGuestNav } from "./components/nav/BasicGuestNav";
import { AdminRoute } from "./components/routing/AdminRoute";
import { ProtectedRoute } from "./components/routing/ProtectedRoute";
import { UserProvider } from "./context/UserContext";
import { AdminAllUsers } from "./pages/AdminAllUsers";
import { AdminUserDetail } from "./pages/AdminUserDetail";
import { ConfirmPage } from "./pages/ConfirmPage";
import { GettingStartedSetup } from "./pages/GettingStartedSetup";
import { HomePage } from "./pages/HomePage";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { Todo } from "./pages/Todo";

export function App() {
  return (
    <UserProvider>
      <BrowserRouter>
        <Routes>
          {/* Public Routes */}
          <Route
            path="/"
            element={
              <GuestLayout>
                <HomePage />
              </GuestLayout>
            }
          />
          <Route
            path="/login"
            element={
              <BasicGuestNav>
                <LoginPage />
              </BasicGuestNav>
            }
          />
          <Route
            path="/register"
            element={
              <BasicGuestNav>
                <RegisterPage />
              </BasicGuestNav>
            }
          />
          <Route
            path="/confirm/:code"
            element={
              <BasicGuestNav>
                <ConfirmPage />
              </BasicGuestNav>
            }
          />

          {/* Signed-In Routes */}
          <Route element={<ProtectedRoute />}>
            <Route
              path="/console/getting-started"
              element={
                <SignedInLayout>
                  <GettingStartedSetup />
                </SignedInLayout>
              }
            />
          </Route>

          {/* Admin-Only Routes */}
          <Route element={<AdminRoute />}>
            <Route
              path="/admin/user"
              element={
                <AdminLayout>
                  <AdminAllUsers />
                </AdminLayout>
              }
            />
            <Route
              path="/admin/user/:id"
              element={
                <AdminLayout>
                  <AdminUserDetail />
                </AdminLayout>
              }
            />
            <Route
              path="/admin/team"
              element={
                <AdminLayout>
                  <Todo title="Admin Teams" />
                </AdminLayout>
              }
            />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </UserProvider>
  );
}

export default App;
