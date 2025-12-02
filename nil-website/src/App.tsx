import { HashRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { Home } from "./pages/Home";
import { KZGDeepDive } from "./pages/KZGDeepDive";
import { ArgonDeepDive } from "./pages/ArgonDeepDive";
import { ShardingDeepDive } from "./pages/ShardingDeepDive";

function App() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="algo/kzg" element={<KZGDeepDive />} />
          <Route path="algo/argon" element={<ArgonDeepDive />} />
          <Route path="algo/sharding" element={<ShardingDeepDive />} />
        </Route>
      </Routes>
    </HashRouter>
  );
}

export default App;
