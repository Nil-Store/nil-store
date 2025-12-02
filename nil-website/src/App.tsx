import { HashRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { Home } from "./pages/Home";
import { Technology } from "./pages/Technology";
import { ScrollToAnchor } from "./components/ScrollToAnchor";

function App() {
  return (
    <HashRouter>
      <ScrollToAnchor />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="technology" element={<Technology />} />
        </Route>
      </Routes>
    </HashRouter>
  );
}

export default App;
