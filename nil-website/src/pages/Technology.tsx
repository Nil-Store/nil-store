import { AlgorithmWalkthrough } from "../components/AlgorithmWalkthrough";
import { motion } from "framer-motion";

export const Technology = () => {
  return (
    <div className="pt-24 pb-12">
      <div className="container mx-auto px-4">
        <div className="text-center mb-16 max-w-3xl mx-auto">
          <motion.h1 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="text-4xl md:text-5xl font-bold mb-6"
          >
            Under the Hood
          </motion.h1>
          <motion.p 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="text-xl text-muted-foreground"
          >
            NilStore combines three cutting-edge cryptographic primitives to create a storage network that is secure, verifiable, and efficient. Explore the interactive demos below to understand the lifecycle of a file.
          </motion.p>
        </div>

        <AlgorithmWalkthrough />
      </div>
    </div>
  );
};
