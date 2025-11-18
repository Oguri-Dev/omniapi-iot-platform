import React from 'react'
import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import '../styles/Dashboard.css'

const Dashboard: React.FC = () => {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="dashboard-container">
      <aside className="sidebar">
        <div className="sidebar-header">
          <h2>🚀 OmniAPI</h2>
          <p className="user-info">
            {user?.full_name || user?.username}
            <span className="user-role">{user?.role}</span>
          </p>
        </div>

        <nav className="sidebar-nav">
          <Link to="/dashboard" className="nav-item">
            <span>📊</span>
            Dashboard
          </Link>
          <Link to="/dashboard/tenants" className="nav-item">
            <span>🏢</span>
            Empresas
          </Link>
          <Link to="/dashboard/sites" className="nav-item">
            <span>🏭</span>
            Centros de Cultivo
          </Link>
          <Link to="/dashboard/external-services" className="nav-item">
            <span>🔌</span>
            Servicios Externos
          </Link>
          <Link to="/dashboard/connectors" className="nav-item">
            <span>🔗</span>
            Conectores
          </Link>
          <Link to="/dashboard/settings" className="nav-item">
            <span>⚙️</span>
            Configuración
          </Link>
        </nav>
        <div className="sidebar-footer">
          <button onClick={handleLogout} className="logout-button">
            🚪 Cerrar Sesión
          </button>
        </div>
      </aside>

      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}

export default Dashboard
