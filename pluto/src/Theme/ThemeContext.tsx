import {
  createContext,
  useContext,
  useState,
  PropsWithChildren,
  useEffect,
  useMemo,
} from "react";
import { Theme, synnaxDark, synnaxLight } from "./theme";
import { applyThemeAsCssVars } from "./css";
import "./theme.css";
import Switch from "../Atoms/Switch/Switch";

const ThemeContext = createContext<{
  theme: Theme;
  toggleTheme: () => void;
}>({
  theme: synnaxLight,
  toggleTheme: () => {
    console.log("unimp")
  },
});

export const useThemeContext = () => useContext(ThemeContext);

export const ThemeProvider = ({
  children,
  themes,
}: PropsWithChildren<{ themes: Theme[] }>) => {
  const [themeIndex, setThemeIndex] = useState<number>(0);

  const toggleTheme = () => {
    console.log("toggleTheme");
    if (themeIndex === themes.length - 1) {
      setThemeIndex(0);
    } else {
      setThemeIndex(themeIndex + 1);
    }
  };

  useEffect(() => {
    applyThemeAsCssVars(document.documentElement, themes[themeIndex]);
  }, [themeIndex]);

  const theme = useMemo(() => themes[themeIndex], [themeIndex]);

  console.log(toggleTheme)

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const ThemeSwitch = () => {
  const { theme, toggleTheme } = useContext(ThemeContext);
  return (
    <Switch
      onChange={() => {
        toggleTheme();
      }}
    ></Switch>
  );
};
