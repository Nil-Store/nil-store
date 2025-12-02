import { motion } from "framer-motion";
import { ShardingDeepDive } from "./ShardingDeepDive";
import { KZGDeepDive } from "./KZGDeepDive";
import { ArgonDeepDive } from "./ArgonDeepDive";

export const Technology = () => {
  return (
    <div className="pt-24 pb-12 container mx-auto px-4 max-w-4xl">
      <div className="mb-24 text-center">
        <motion.h1 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl md:text-6xl font-bold mb-6"
        >
          Under the Hood
        </motion.h1>
        <motion.p 
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="text-xl text-muted-foreground max-w-2xl mx-auto"
        >
          NilStore combines three cutting-edge cryptographic primitives to create a storage network that is secure, verifiable, and efficient. 
        </motion.p>
      </div>

      <div className="space-y-32">
        <section id="sharding" className="scroll-mt-32">
          <ShardingDeepDive />
        </section>

        <div className="w-full border-t border-dashed border-muted-foreground/20" />

        <section id="kzg" className="scroll-mt-32">
          <KZGDeepDive />
        </section>

        <div className="w-full border-t border-dashed border-muted-foreground/20" />

        <section id="sealing" className="scroll-mt-32">
          <ArgonDeepDive />
        </section>
      </div>
    </div>
  );
};