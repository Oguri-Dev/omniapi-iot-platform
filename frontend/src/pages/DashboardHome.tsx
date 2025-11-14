import React from 'react'
import { useAuth } from '../contexts/AuthContext'

const DashboardHome: React.FC = () => {
  const { user } = useAuth()

  return (
    <div className="dashboard-home">
      <h1>Bienvenido, {user?.full_name || user?.username}!</h1>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon">🔌</div>
          <div className="stat-content">
            <h3>Servicios Activos</h3>
            <p className="stat-number">0</p>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">🔗</div>
          <div className="stat-content">
            <h3>Conectores</h3>
            <p className="stat-number">0</p>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-content">
            <h3>Datos Procesados</h3>
            <p className="stat-number">0</p>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">⚡</div>
          <div className="stat-content">
            <h3>Estado del Sistema</h3>
            <p className="stat-status">✅ Operativo</p>
          </div>
        </div>
      </div>

      <div className="quick-actions">
        <h2>Acciones Rápidas</h2>
        <div className="actions-grid">
          <button className="action-button">
            <span>➕</span>
            Nuevo Servicio
          </button>
          <button className="action-button">
            <span>🔄</span>
            Sincronizar Datos
          </button>
          <button className="action-button">
            <span>📈</span>
            Ver Métricas
          </button>
          <button className="action-button">
            <span>⚙️</span>
            Configuración
          </button>
        </div>
      </div>
    </div>
  )
}

export default DashboardHome
