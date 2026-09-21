"use client";

import { motion } from "framer-motion";

const stages = [
  { label: "Upload", detail: "PDF, DOCX or TXT" },
  { label: "Extract", detail: "Text + structure" },
  { label: "Validate", detail: "Schema-checked" },
  { label: "Match", detail: "Scored vs. job" },
  { label: "Interview", detail: "Grounded questions" },
];

/**
 * A deliberately restrained SVG pipeline diagram rather than a 3D/Three.js
 * scene: it's the same conceptual "AI processing pipeline" moment the brief
 * asks for, renders identically everywhere, costs nothing on low-power
 * devices, and can't visually break. One orchestrated reveal animation on
 * scroll-into-view, not per-element hover effects everywhere.
 */
export function Pipeline() {
  return (
    <div className="relative overflow-hidden rounded-card border border-line bg-navy-950 p-8 md:p-10">
      <div className="mb-8 flex items-center justify-between">
        <span className="text-xs font-medium uppercase tracking-wide text-white/40">Resume processing pipeline</span>
        <span className="flex h-2 w-2 rounded-full bg-amber-400" />
      </div>

      <div className="relative flex flex-col gap-0 md:flex-row md:items-center md:justify-between">
        <motion.div
          className="absolute left-4 right-4 top-4 hidden h-px bg-white/10 md:block"
          initial={{ scaleX: 0 }}
          whileInView={{ scaleX: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 1.1, ease: "easeOut" }}
          style={{ transformOrigin: "left" }}
        />

        {stages.map((stage, i) => (
          <motion.div
            key={stage.label}
            className="relative flex flex-1 flex-col items-start gap-3 py-3 md:items-center md:py-0"
            initial={{ opacity: 0, y: 12 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ duration: 0.5, delay: i * 0.12 }}
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-full border border-white/20 bg-navy-900 text-xs font-semibold text-amber-400 md:mx-auto">
              {i + 1}
            </div>
            <div className="md:text-center">
              <div className="text-sm font-semibold text-white">{stage.label}</div>
              <div className="text-xs text-white/50">{stage.detail}</div>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
