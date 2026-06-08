/**
 * main.tsx — Application entry point.
 *
 * Creates the React 18 root, wraps the app in BrowserRouter for
 * client-side routing, and mounts to the #root div.
 */

import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';

// Global styles (Tailwind directives + custom base/component classes)
import './index.css';
import 'animate.css';

// Locate the mount point
const rootElement = document.getElementById('root');

if (!rootElement) {
  throw new Error(
    '[CodeTasker] Could not find #root element. ' +
    'Make sure index.html contains <div id="root"></div>.'
  );
}

// Create and render the React 18 root
createRoot(rootElement).render(
  <BrowserRouter>
    <App />
  </BrowserRouter>
);
