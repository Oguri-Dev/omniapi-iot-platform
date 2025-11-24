import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import ProtectedRoute from './components/ProtectedRoute'
import Login from './pages/Login'
import Setup from './pages/Setup'
import Dashboard from './pages/Dashboard'
import DashboardHome from './pages/DashboardHome'
import Services from './pages/Services'
import Builder from './pages/Builder'
import Connectors from './pages/Connectors'
import Tenants from './pages/Tenants'
import Sites from './pages/Sites'
import ExternalServices from './pages/ExternalServices'
import './App.css'

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/setup" element={<Setup />} />
          <Route path="/login" element={<Login />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          >
            <Route index element={<DashboardHome />} />
            <Route path="tenants" element={<Tenants />} />
            <Route path="sites" element={<Sites />} />
            <Route path="external-services" element={<ExternalServices />} />
            <Route path="services" element={<Services />} />
            <Route path="connectors" element={<Connectors />} />
            <Route path="builder" element={<Builder />} />
            <Route path="settings" element={<div>Configuración (próximamente)</div>} />
          </Route>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="*" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
