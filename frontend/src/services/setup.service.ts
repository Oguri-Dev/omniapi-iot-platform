import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3000'

// Verificar si el sistema necesita configuración inicial
export const checkSetupStatus = async (): Promise<boolean> => {
  try {
    const response = await axios.get(`${API_URL}/api/auth/setup/check`)
    return response.data.data?.needsSetup || false
  } catch (error) {
    console.error('Error checking setup status:', error)
    return false
  }
}
