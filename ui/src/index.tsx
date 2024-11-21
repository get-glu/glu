import { createRoot } from 'react-dom/client';
import { Provider } from 'react-redux';
import { App } from './App';
import { store } from './store';
import React from 'react';

const container = document.getElementById('app') as HTMLElement;
const root = createRoot(container);
root.render(
  // <React.StrictMode>
  <Provider store={store}>
    <App />
  </Provider>
  //  </React.StrictMode>
);
