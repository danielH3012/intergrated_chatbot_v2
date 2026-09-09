import { createContext, useContext, useEffect, useState } from 'react'
import { getStoredUser } from './services.js'

const AuthContext = createContext(null)

/**
 * AuthProvider — wraps the app and provides the current authenticated user.
 * Accepts an optional `user` prop from the parent so the live authenticated
 * user is passed directly (avoids stale localStorage reads).
 */
export function AuthProvider({ children, user: userProp }) {
  const [user, setUser] = useState(() => userProp ?? getStoredUser())

  // Sync whenever the parent's user prop changes (login / logout)
  useEffect(() => {
    if (userProp !== undefined) setUser(userProp)
  }, [userProp])

  useEffect(() => {
    function handleUnauthorized() {
      setUser(null)
    }
    window.addEventListener('qtera:unauthorized', handleUnauthorized)
    return () => window.removeEventListener('qtera:unauthorized', handleUnauthorized)
  }, [])

  return (
    <AuthContext.Provider value={{ user, setUser }}>
      {children}
    </AuthContext.Provider>
  )
}

/**
 * useAuth() — returns { user, setUser } from the nearest AuthProvider.
 */
export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an <AuthProvider>')
  }
  return ctx
}
