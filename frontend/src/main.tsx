import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import { ThemeProvider } from './components/ThemeContext'
import App from './App'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <ThemeProvider>
            <App/>
        </ThemeProvider>
    </React.StrictMode>
)
