import { motion } from "framer-motion";
import { File, Lock, CheckCircle, Server, ArrowRight } from "lucide-react";
import { Link } from "react-router-dom";
import { cn } from "../lib/utils";

const steps = [
  {
    id: 1,
    title: "Sharding & Encoding",
    description: "The user's file is split into 1KB chunks. Each chunk is mapped to a field element (Fr) using SHA256.",
    icon: <File className="w-6 h-6" />,
    link: "/technology/sharding",
    linkText: "Deep Dive: Sharding",
    visual: (
      <div className="grid grid-cols-4 gap-2">
        {[...Array(16)].map((_, i) => (
          <motion.div
            initial={{ opacity: 0 }}
            whileInView={{ opacity: 1 }}
            viewport={{ once: true }}
            transition={{ delay: i * 0.05 }}
            key={i}
            className="w-12 h-12 bg-blue-500/20 border border-blue-500 rounded flex items-center justify-center text-xs"
          >
            1KB
          </motion.div>
        ))}
      </div>
    )
  },
  {
    id: 2,
    title: "KZG Commitment",
    description: "Chunks are packed into a polynomial. A KZG commitment (48 bytes) is generated, representing the entire dataset compactly.",
    icon: <Lock className="w-6 h-6" />,
    link: "/technology/kzg",
    linkText: "Deep Dive: KZG",
    visual: (
      <div className="relative w-48 h-48 bg-purple-500/10 rounded-full flex items-center justify-center border-2 border-dashed border-purple-500 animate-spin-slow">
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="font-mono text-purple-500 font-bold">C(x)</span>
        </div>
      </div>
    )
  },
  {
    id: 3,
    title: "Argon2id Sealing",
    description: "Storage nodes must 'seal' the data using a memory-hard function (Argon2id). This takes time (~191ms/KB), proving they aren't generating it on the fly.",
    icon: <Server className="w-6 h-6" />,
    link: "/technology/sealing",
    linkText: "Deep Dive: Sealing",
    visual: (
      <div className="flex gap-4 items-center">
        <div className="w-24 h-24 bg-blue-500 rounded-lg flex items-center justify-center text-white font-bold">Data</div>
        <motion.div 
          animate={{ x: [0, 10, 0] }} 
          transition={{ repeat: Infinity, duration: 1.5 }}
          className="text-2xl text-muted-foreground"
        >→</motion.div>
        <div className="w-24 h-24 bg-red-500 rounded-lg flex items-center justify-center text-white font-bold shadow-lg shadow-red-500/20">Sealed</div>
      </div>
    )
  },
  {
    id: 4,
    title: "Proof Verification",
    description: "The network challenges a node. The node provides a proof for a random chunk. Verifiers check this in < 1ms.",
    icon: <CheckCircle className="w-6 h-6" />,
    visual: (
      <div className="flex flex-col gap-4 items-center">
        <div className="text-green-500 text-6xl">✓</div>
        <div className="font-mono bg-black text-green-400 p-4 rounded-lg text-sm">
          {`verify(C, z, y, proof) == true`}
          <br/>
          <span className="text-gray-500">Time: 0.94ms</span>
        </div>
      </div>
    )
  },
];

export const AlgorithmWalkthrough = () => {
  return (
    <section className="py-12">
      <div className="container mx-auto">
        <div className="space-y-24">
          {steps.map((step, index) => (
            <motion.div
              key={step.id}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-100px" }}
              transition={{ duration: 0.6 }}
              className="flex flex-col lg:flex-row gap-12 items-center"
            >
              {/* Text Side */}
              <div className={cn("lg:w-1/2", index % 2 === 1 && "lg:order-2")}>
                <div className="flex items-center gap-4 mb-6">
                  <div className="p-3 bg-primary/10 rounded-2xl text-primary">
                    {step.icon}
                  </div>
                  <div className="text-sm font-bold text-primary uppercase tracking-wider">Step 0{step.id}</div>
                </div>
                
                <h3 className="text-3xl font-bold mb-6">{step.title}</h3>
                <p className="text-xl text-muted-foreground mb-8 leading-relaxed">
                  {step.description}
                </p>

                {step.link && (
                  <Link 
                    to={step.link}
                    className="inline-flex items-center gap-2 text-primary font-medium hover:gap-3 transition-all"
                  >
                    {step.linkText} <ArrowRight className="w-4 h-4" />
                  </Link>
                )}
              </div>

              {/* Visual Side */}
              <div className={cn(
                "lg:w-1/2 w-full bg-secondary/30 rounded-3xl p-12 flex items-center justify-center min-h-[400px] border",
                index % 2 === 1 && "lg:order-1"
              )}>
                {step.visual}
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
};
