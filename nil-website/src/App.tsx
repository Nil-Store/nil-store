import { HashRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { Home } from "./pages/Home";
import { Technology } from "./pages/Technology";
import { KZGDeepDive } from "./pages/KZGDeepDive";
import { ArgonDeepDive } from "./pages/ArgonDeepDive";
import { ShardingDeepDive } from "./pages/ShardingDeepDive";

function App() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="technology">
            <Route index element={<Technology />} />
            <Route path="sharding" element={<ShardingDeepDive />} />
            <Route path="kzg" element={<KZGDeepDive />} />
            <Route path="sealing" element={<ArgonDeepDive />} />
          </Route>
        </Route>
      </Routes>
    </HashRouter>
  );
}

export default App;
