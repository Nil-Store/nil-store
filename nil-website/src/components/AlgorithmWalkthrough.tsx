import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { File, Lock, CheckCircle, Server } from "lucide-react";
import { Link } from "react-router-dom";
import { cn } from "../lib/utils";

const steps = [
  {
    id: 1,
    title: "Sharding & Encoding",
    description: "The user's file is split into 1KB chunks. Each chunk is mapped to a field element (Fr) using SHA256.",
    icon: <File className="w-6 h-6" />,
  },
  {
    id: 2,
    title: "KZG Commitment",
    description: "Chunks are packed into a polynomial. A KZG commitment (48 bytes) is generated, representing the entire dataset compactly.",
    icon: <Lock className="w-6 h-6" />,
  },
  {
    id: 3,
    title: "Argon2id Sealing",
    description: "Storage nodes must 'seal' the data using a memory-hard function (Argon2id). This takes time (~191ms/KB), proving they aren't generating it on the fly.",
    icon: <Server className="w-6 h-6" />,
  },
  {
    id: 4,
    title: "Proof Verification",
    description: "The network challenges a node. The node provides a proof for a random chunk. Verifiers check this in < 1ms.",
    icon: <CheckCircle className="w-6 h-6" />,
  },
];

export const AlgorithmWalkthrough = () => {
  const [activeStep, setActiveStep] = useState(1);

  return (
    <section className="py-24">
      <div className="container mx-auto px-4">
        <div className="text-center mb-16">
          <h2 className="text-3xl font-bold mb-4">How It Works</h2>
          <p className="text-muted-foreground">Step-by-step journey of a file in the NilStore network.</p>
        </div>

        <div className="flex flex-col lg:flex-row gap-12">
          {/* Steps List */}
          <div className="lg:w-1/3 space-y-4">
            {steps.map((step) => (
              <button
                key={step.id}
                onClick={() => setActiveStep(step.id)}
                className={cn(
                  "w-full text-left p-6 rounded-xl border transition-all duration-300 flex items-center gap-4",
                  activeStep === step.id
                    ? "bg-primary text-primary-foreground border-primary ring-2 ring-primary/20"
                    : "bg-card hover:bg-secondary/50"
                )}
              >
                <div className={cn(
                  "p-2 rounded-full",
                  activeStep === step.id ? "bg-primary-foreground/20" : "bg-secondary"
                )}>
                  {step.icon}
                </div>
                <div>
                  <h3 className="font-semibold">{step.title}</h3>
                </div>
              </button>
            ))}
          </div>

          {/* Visualization Area */}
          <div className="lg:w-2/3 bg-secondary/20 rounded-3xl p-8 lg:p-12 flex items-center justify-center min-h-[400px] relative overflow-hidden">
            <AnimatePresence mode="wait">
              <motion.div
                key={activeStep}
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 1.1 }}
                transition={{ duration: 0.3 }}
                className="text-center max-w-lg"
              >
                <div className="mb-8 flex justify-center">
                  {activeStep === 1 && (
                    <div className="grid grid-cols-4 gap-2">
                      {[...Array(16)].map((_, i) => (
                        <motion.div
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          transition={{ delay: i * 0.05 }}
                          key={i}
                          className="w-12 h-12 bg-blue-500/20 border border-blue-500 rounded flex items-center justify-center text-xs"
                        >
                          1KB
                        </motion.div>
                      ))}
                    </div>
                  )}
                  {activeStep === 2 && (
                    <div className="relative w-48 h-48 bg-purple-500/10 rounded-full flex items-center justify-center border-2 border-dashed border-purple-500 animate-spin-slow">
                      <div className="absolute inset-0 flex items-center justify-center">
                        <span className="font-mono text-purple-500 font-bold">C(x)</span>
                      </div>
                    </div>
                  )}
                  {activeStep === 3 && (
                    <div className="flex gap-4 items-center">
                      <div className="w-24 h-24 bg-blue-500 rounded-lg flex items-center justify-center text-white font-bold">Data</div>
                      <motion.div 
                        animate={{ x: [0, 10, 0] }} 
                        transition={{ repeat: Infinity, duration: 1.5 }}
                        className="text-2xl text-muted-foreground"
                      >→</motion.div>
                      <div className="w-24 h-24 bg-red-500 rounded-lg flex items-center justify-center text-white font-bold shadow-lg shadow-red-500/20">Sealed</div>
                    </div>
                  )}
                  {activeStep === 4 && (
                    <div className="flex flex-col gap-4 items-center">
                      <div className="text-green-500 text-6xl">✓</div>
                      <div className="font-mono bg-black text-green-400 p-4 rounded-lg text-sm">
                        {`verify(C, z, y, proof) == true`}
                        <br/>
                        <span className="text-gray-500">Time: 0.94ms</span>
                      </div>
                    </div>
                  )}
                </div>
                <h3 className="text-2xl font-bold mb-4">{steps[activeStep - 1].title}</h3>
                <p className="text-lg text-muted-foreground mb-6">{steps[activeStep - 1].description}</p>
                
                {activeStep === 1 && (
                  <Link to="/algo/sharding" className="inline-block px-6 py-2 bg-primary text-primary-foreground rounded-full text-sm font-medium hover:opacity-90">
                    Deep Dive: Sharding
                  </Link>
                )}
                {activeStep === 2 && (
                  <Link to="/algo/kzg" className="inline-block px-6 py-2 bg-primary text-primary-foreground rounded-full text-sm font-medium hover:opacity-90">
                    Deep Dive: KZG
                  </Link>
                )}
                {activeStep === 3 && (
                  <Link to="/algo/argon" className="inline-block px-6 py-2 bg-primary text-primary-foreground rounded-full text-sm font-medium hover:opacity-90">
                    Deep Dive: Sealing
                  </Link>
                )}
              </motion.div>
            </AnimatePresence>
          </div>
        </div>
      </div>
    </section>
  );
};
