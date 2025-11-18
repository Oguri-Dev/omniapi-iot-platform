import React, { createContext, useContext, useState, useEffect } from 'react'
import type { ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import authService from '../services/auth.service'
import { checkSetupStatus } from '../services/setup.service'
import type { User } from '../services/auth.service'

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  loading: boolean
  needsSetup: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: ReactNode
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [needsSetup, setNeedsSetup] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    const checkAuth = async () => {
      // Primero verificar si el sistema necesita setup
      const setupRequired = await checkSetupStatus()
      setNeedsSetup(setupRequired)

      if (setupRequired) {
        // Si necesita setup, redirigir a la página de setup
        navigate('/setup')
        setLoading(false)
        return
      }

      // Si no necesita setup, verificar autenticación normal
      const currentUser = authService.getCurrentUser()
      setUser(currentUser)
      setLoading(false)
    }

    checkAuth()
  }, [navigate])

  const login = async (username: string, password: string) => {
    try {
      const response = await authService.login({ username, password })
      setUser(response.data.user)
    } catch (error) {
      throw error
    }
  }

  const logout = () => {
    authService.logout()
    setUser(null)
  }

  const value = {
    user,
    isAuthenticated: !!user,
    login,
    logout,
    loading,
    needsSetup,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
