import { Link, Outlet, useLocation } from "react-router-dom";
import { motion } from "framer-motion";
import { Layers, File, Lock, ShieldAlert, ChevronRight } from "lucide-react";
import { cn } from "../lib/utils";
import { TechnologyProvider, useTechnology } from "../context/TechnologyContext";

const sidebarItems = [
  {
    title: "Protocol Overview",
    path: "/technology",
    exact: true,
    icon: <Layers className="w-4 h-4" />,
    step: "Intro"
  },
  {
    title: "Data Sharding",
    path: "/technology/sharding",
    icon: <File className="w-4 h-4" />,
    step: "Step 1"
  },
  {
    title: "KZG Commitments",
    path: "/technology/kzg",
    icon: <Lock className="w-4 h-4" />,
    step: "Step 2"
  },
  {
    title: "Proof of Seal",
    path: "/technology/sealing",
    icon: <ShieldAlert className="w-4 h-4" />,
    step: "Step 3"
  },
];

const Sidebar = () => {
  const location = useLocation();
  const { highlightedPath } = useTechnology();

  return (
    <aside className="lg:w-72 flex-shrink-0">
      <div className="sticky top-24 bg-card border rounded-2xl p-4 shadow-sm">
        <h2 className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-4 px-2">
          The NilStore Pipeline
        </h2>
        <nav className="space-y-2">
          {sidebarItems.map((item) => {
            const isRouteActive = item.exact 
              ? location.pathname === item.path
              : location.pathname.startsWith(item.path);
            
            const isHighlighted = highlightedPath === item.path;
            const isActive = isRouteActive || isHighlighted;

            return (
              <Link
                key={item.path}
                to={item.path}
                className={cn(
                  "flex items-center gap-3 px-3 py-3 text-sm rounded-xl transition-all relative group overflow-hidden",
                  isActive 
                    ? "bg-primary text-primary-foreground shadow-md" 
                    : "hover:bg-secondary text-muted-foreground hover:text-foreground"
                )}
              >
                {isActive && (
                  <motion.div
                    layoutId="active-tech-bg"
                    className="absolute inset-0 bg-primary rounded-xl -z-10"
                    transition={{ type: "spring", bounce: 0.2, duration: 0.6 }}
                  />
                )}
                
                <div className={cn(
                  "flex items-center justify-center w-8 h-8 rounded-lg transition-colors",
                  isActive ? "bg-white/20" : "bg-secondary group-hover:bg-background"
                )}>
                  {item.icon}
                </div>
                
                <div className="flex-1">
                  <div className={cn("text-[10px] uppercase font-bold opacity-70", isActive ? "text-primary-foreground" : "text-muted-foreground")}>
                    {item.step}
                  </div>
                  <div className="font-semibold">{item.title}</div>
                </div>

                {isActive && (
                  <motion.div
                    initial={{ opacity: 0, x: -5 }}
                    animate={{ opacity: 1, x: 0 }}
                    className="mr-1"
                  >
                    <ChevronRight className="w-4 h-4 opacity-50" />
                  </motion.div>
                )}
              </Link>
            );
          })}
        </nav>

        <div className="mt-8 pt-6 border-t px-2">
          <p className="mb-3 text-xs font-semibold text-foreground">Resources</p>
          <ul className="space-y-2">
            <li>
              <a href="#" className="text-xs text-muted-foreground hover:text-primary flex items-center gap-2 transition-colors group">
                <div className="w-1 h-1 rounded-full bg-muted-foreground group-hover:bg-primary transition-colors" />
                Full Whitepaper
              </a>
            </li>
            <li>
              <a href="#" className="text-xs text-muted-foreground hover:text-primary flex items-center gap-2 transition-colors group">
                <div className="w-1 h-1 rounded-full bg-muted-foreground group-hover:bg-primary transition-colors" />
                Rust SDK Docs
              </a>
            </li>
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
      <div className="container mx-auto px-4 pt-24 pb-12 flex flex-col lg:flex-row gap-8 lg:gap-12 min-h-screen">
        <Sidebar />
        <div className="flex-1 min-w-0">
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, ease: "easeOut" }}
          >
            <Outlet />
          </motion.div>
        </div>
      </div>
    </TechnologyProvider>
  );
};