import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { AuthProvider } from './context/AuthContext.tsx'
import { BillingProvider } from './context/BillingContext'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
     <AuthProvider>
      <BillingProvider>
        <App />
      </BillingProvider>
    </AuthProvider>
  </StrictMode>,
)
