import axios from 'axios'

// All API calls go through this instance
// Base URL comes from .env (VITE_API_BASE_URL)
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  withCredentials: true, // sends httpOnly cookie with every request
  headers: {
    'Content-Type': 'application/json',
  },
})

// Response interceptor — if 401, redirect to login
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      window.location.href = '/auth'
    }
    return Promise.reject(error)
  },
)

export default api