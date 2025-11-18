import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import axios from 'axios'
import '../styles/Setup.css'

interface SetupFormData {
  username: string
  email: string
  password: string
  confirmPassword: string
  fullName: string
}

const Setup: React.FC = () => {
  const navigate = useNavigate()
  const [formData, setFormData] = useState<SetupFormData>({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    fullName: '',
  })
  const [error, setError] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value,
    })
    setError('')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    // Validaciones
    if (!formData.username || !formData.email || !formData.password) {
      setError('Todos los campos obligatorios deben completarse')
      return
    }

    if (formData.password.length < 6) {
      setError('La contraseña debe tener al menos 6 caracteres')
      return
    }

    if (formData.password !== formData.confirmPassword) {
      setError('Las contraseñas no coinciden')
      return
    }

    if (!formData.email.includes('@')) {
      setError('Email inválido')
      return
    }

    setLoading(true)

    try {
      const response = await axios.post(`${import.meta.env.VITE_API_URL}/api/auth/setup`, {
        username: formData.username,
        email: formData.email,
        password: formData.password,
        fullName: formData.fullName || formData.username,
      })

      if (response.data.success) {
        // Redirigir al login
        navigate('/login', {
          state: {
            message: 'Administrador creado exitosamente. Ya puedes iniciar sesión.',
          },
        })
      } else {
        setError(response.data.message || 'Error al crear el administrador')
      }
    } catch (err: any) {
      setError(
        err.response?.data?.message || 'Error al crear el administrador. Inténtalo de nuevo.'
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="setup-container">
      <div className="setup-card">
        <div className="setup-header">
          <div className="setup-icon">🚀</div>
          <h1>Configuración Inicial</h1>
          <p>Crea el primer usuario administrador del sistema</p>
        </div>

        {error && (
          <div className="error-message">
            <span>⚠️</span> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="setup-form">
          <div className="form-group">
            <label htmlFor="username">
              Nombre de Usuario <span className="required">*</span>
            </label>
            <input
              type="text"
              id="username"
              name="username"
              value={formData.username}
              onChange={handleChange}
              placeholder="admin"
              required
              autoFocus
            />
          </div>

          <div className="form-group">
            <label htmlFor="email">
              Email <span className="required">*</span>
            </label>
            <input
              type="email"
              id="email"
              name="email"
              value={formData.email}
              onChange={handleChange}
              placeholder="admin@omniapi.com"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="fullName">Nombre Completo (opcional)</label>
            <input
              type="text"
              id="fullName"
              name="fullName"
              value={formData.fullName}
              onChange={handleChange}
              placeholder="Administrador del Sistema"
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">
              Contraseña <span className="required">*</span>
            </label>
            <input
              type="password"
              id="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              placeholder="••••••••"
              required
              minLength={6}
            />
            <small>Mínimo 6 caracteres</small>
          </div>

          <div className="form-group">
            <label htmlFor="confirmPassword">
              Confirmar Contraseña <span className="required">*</span>
            </label>
            <input
              type="password"
              id="confirmPassword"
              name="confirmPassword"
              value={formData.confirmPassword}
              onChange={handleChange}
              placeholder="••••••••"
              required
              minLength={6}
            />
          </div>

          <button type="submit" className="setup-button" disabled={loading}>
            {loading ? (
              <>
                <span className="spinner"></span> Creando...
              </>
            ) : (
              <>
                <span>🔐</span> Crear Administrador
              </>
            )}
          </button>
        </form>

        <div className="setup-info">
          <p>
            <strong>⚠️ Importante:</strong> Este usuario tendrá acceso completo al sistema.
            Asegúrate de usar una contraseña segura.
          </p>
        </div>
      </div>
    </div>
  )
}

export default Setup
