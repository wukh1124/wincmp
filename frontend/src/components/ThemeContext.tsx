import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';

export type ThemeId = 'carbon' | 'cream' | 'sketch';

interface ThemeMeta {
  id: ThemeId;
  name: string;
  nameZh: string;
  description: string;
}

export const THEMES: ThemeMeta[] = [
  { id: 'carbon', name: 'Carbon', nameZh: 'Carbon', description: 'Dark monospace brutalism' },
  { id: 'cream', name: 'Cream', nameZh: '奶油暖白', description: 'Warm cream light' },
  { id: 'sketch', name: 'Sketch', nameZh: 'Sketch 線稿', description: 'Wireframe on graph paper' },
];

export type FontSizeId = 'small' | 'medium' | 'large';

interface FontSizeMeta {
  id: FontSizeId;
  name: string;
  nameZh: string;
}

export const FONT_SIZES: FontSizeMeta[] = [
  { id: 'small', name: 'Small', nameZh: '小' },
  { id: 'medium', name: 'Medium', nameZh: '中' },
  { id: 'large', name: 'Large', nameZh: '大' },
];

interface ThemeContextValue {
  theme: ThemeId;
  setTheme: (id: ThemeId) => void;
  themes: ThemeMeta[];
  fontSize: FontSizeId;
  setFontSize: (id: FontSizeId) => void;
  fontSizes: FontSizeMeta[];
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: 'sketch',
  setTheme: () => {},
  themes: THEMES,
  fontSize: 'large',
  setFontSize: () => {},
  fontSizes: FONT_SIZES,
});

const STORAGE_KEY = 'wincmp-theme';
const FONT_SIZE_STORAGE_KEY = 'wincmp-font-size';

function getInitialTheme(): ThemeId {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && ['carbon', 'cream', 'sketch'].includes(stored)) {
      return stored as ThemeId;
    }
    if (stored === 'xai') {
      return 'sketch';
    }
  } catch {}
  return 'sketch';
}

function getInitialFontSize(): FontSizeId {
  try {
    const stored = localStorage.getItem(FONT_SIZE_STORAGE_KEY);
    if (stored && ['small', 'medium', 'large'].includes(stored)) {
      return stored as FontSizeId;
    }
  } catch {}
  return 'large'; /* 預設大小為大 */
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<ThemeId>(getInitialTheme);
  const [fontSize, setFontSizeState] = useState<FontSizeId>(getInitialFontSize);

  const setTheme = useCallback((id: ThemeId) => {
    setThemeState(id);
    try {
      localStorage.setItem(STORAGE_KEY, id);
    } catch {}
    document.documentElement.setAttribute('data-theme', id);
  }, []);

  const setFontSize = useCallback((id: FontSizeId) => {
    setFontSizeState(id);
    try {
      localStorage.setItem(FONT_SIZE_STORAGE_KEY, id);
    } catch {}
    document.documentElement.setAttribute('data-font-size', id);
  }, []);

  // 套用主題與字型大小
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  useEffect(() => {
    document.documentElement.setAttribute('data-font-size', fontSize);
  }, [fontSize]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, themes: THEMES, fontSize, setFontSize, fontSizes: FONT_SIZES }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}
