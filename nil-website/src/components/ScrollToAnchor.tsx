import { useEffect } from "react";
import { useLocation } from "react-router-dom";

export function ScrollToAnchor() {
  const { hash } = useLocation();

  useEffect(() => {
    if (hash) {
      const element = document.getElementById(hash.slice(1));
      if (element) {
        element.scrollIntoView({ behavior: "smooth" });
      }
    }
  }, [hash]);

  return null;
}
