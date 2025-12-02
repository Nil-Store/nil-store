import { Link, Outlet, useLocation } from "react-router-dom";
import { motion } from "framer-motion";
import { Layers, File, Lock, ShieldAlert, ChevronRight } from "lucide-react";
import { cn } from "../lib/utils";
import { TechnologyProvider, useTechnology } from "../context/TechnologyContext";

const sidebarItems = [
  {
    title: "Overview",
    path: "/technology",
    exact: true,
    icon: <Layers className="w-4 h-4" />,
  },
  {
    title: "Data Sharding",
    path: "/technology/sharding",
    icon: <File className="w-4 h-4" />,
  },
  {
    title: "KZG Commitments",
    path: "/technology/kzg",
    icon: <Lock className="w-4 h-4" />,
  },
  {
    title: "Proof of Seal",
    path: "/technology/sealing",
    icon: <ShieldAlert className="w-4 h-4" />,
  },
];

const Sidebar = () => {
  const location = useLocation();
  const { highlightedPath } = useTechnology();

  return (
    <aside className="lg:w-64 flex-shrink-0">
      <div className="sticky top-24">
        <h2 className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-4 px-4">
          Core Concepts
        </h2>
        <nav className="space-y-1">
          {sidebarItems.map((item) => {
            const isRouteActive = item.exact 
              ? location.pathname === item.path
              : location.pathname.startsWith(item.path);
            
            // Highlight if either the route is active OR if it's being hovered in the walkthrough
            const isHighlighted = highlightedPath === item.path;
            const isActive = isRouteActive || isHighlighted;

            return (
              <Link
                key={item.path}
                to={item.path}
                className={cn(
                  "flex items-center gap-3 px-4 py-3 text-sm font-medium rounded-lg transition-all relative group",
                  isActive 
                    ? "text-primary bg-primary/10" 
                    : "text-muted-foreground hover:text-foreground hover:bg-secondary/50"
                )}
              >
                {item.icon}
                {item.title}
                {isActive && (
                  <motion.div
                    layoutId="active-tech-nav"
                    className="absolute left-0 top-0 bottom-0 w-1 bg-primary rounded-full"
                    transition={{ type: "spring", stiffness: 300, damping: 30 }}
                  />
                )}
              </Link>
            );
          })}
        </nav>

        <div className="mt-8 p-4 bg-secondary/20 rounded-xl border text-xs text-muted-foreground">
          <p className="mb-2 font-semibold text-foreground">Developer Resources</p>
          <ul className="space-y-2">
            <li><a href="#" className="hover:underline flex items-center gap-1">API Reference <ChevronRight className="w-3 h-3" /></a></li>
            <li><a href="#" className="hover:underline flex items-center gap-1">Rust SDK <ChevronRight className="w-3 h-3" /></a></li>
          </ul>
        </div>
      </div>
    </aside>
  );
};

export const TechnologyLayout = () => {
  const location = useLocation();

  return (
    <TechnologyProvider>
      <div className="container mx-auto px-4 pt-24 pb-12 flex flex-col lg:flex-row gap-12 min-h-screen">
        <Sidebar />
        <div className="flex-1 min-w-0">
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Outlet />
          </motion.div>
        </div>
      </div>
    </TechnologyProvider>
  );
};
