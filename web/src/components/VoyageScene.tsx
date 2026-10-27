// SPDX-License-Identifier: AGPL-3.0-or-later
// "Le voyage": every project travels in its own vehicle, all through the same porthole,
// and the rocket is the company. Same scene as the top of the landing page (Voyage.dc.html).
import { useId, type CSSProperties } from 'react'

const INK = '#111111'

function Hub({ x, y, size, color }: { x: number; y: number; size: number; color: string }) {
  return <use href="#hub" x={x} y={y} width={size} height={size} style={{ color } as CSSProperties} />
}

/** Decorative; needs <HubSymbolDefs /> on the page. */
export function VoyageScene({ className }: { className?: string }) {
  const clip = useId()
  return (
    <svg aria-hidden="true" viewBox="0 0 1400 560" width="100%" className={className} style={{ display: 'block', background: '#A8DBFF' }}>
      <defs>
        <clipPath id={clip}>
          <rect x="0" y="0" width="1400" height="560" rx="26" />
        </clipPath>
      </defs>
      <g clipPath={`url(#${clip})`}>
        <path d="M60 90h70a18 18 0 000-36a24 24 0 00-46 4a16 16 0 00-24 32z" fill="#FFFFFF" stroke={INK} strokeWidth="3" />
        <path d="M940 120h80a18 18 0 000-36a26 26 0 00-50 4a18 18 0 00-30 32z" fill="#FFFFFF" stroke={INK} strokeWidth="3" />
        <circle cx="380" cy="80" r="34" fill="#FFE066" stroke={INK} strokeWidth="3.5" />
        <path d="M140 300C330 160 640 120 900 150S1120 200 1170 170" stroke={INK} strokeWidth="3" strokeDasharray="3 12" strokeLinecap="round" fill="none" />

        <rect x="-10" y="370" width="440" height="200" fill="#6FB6E8" stroke={INK} strokeWidth="3" />
        <path d="M30 400q15-8 30 0t30 0M250 420q15-8 30 0t30 0M90 520q15-8 30 0t30 0" stroke={INK} strokeWidth="3" fill="none" strokeLinecap="round" />
        <rect x="430" y="370" width="980" height="200" fill="#7FD8B6" stroke={INK} strokeWidth="3" />

        {/* Boat */}
        <g stroke={INK} strokeWidth="3.5" strokeLinejoin="round">
          <rect x="196" y="236" width="28" height="44" fill="#FFE066" />
          <rect x="196" y="244" width="28" height="9" fill={INK} />
          <rect x="130" y="272" width="170" height="44" rx="6" fill="#FFFFFF" />
          <rect x="160" y="250" width="100" height="26" rx="6" fill="#FFFFFF" />
          <path d="M60 314H360L332 372H92Z" fill="#B9AEFF" />
        </g>
        <Hub x={146} y={276} size={36} color="#B9AEFF" />
        <Hub x={198} y={276} size={36} color="#B9AEFF" />
        <Hub x={250} y={276} size={36} color="#B9AEFF" />

        {/* Submarine */}
        <g stroke={INK} strokeWidth="3.5" strokeLinejoin="round">
          <path d="M226 446v-34h10" fill="none" />
          <rect x="196" y="440" width="54" height="30" rx="8" fill="#FFE066" />
          <rect x="110" y="462" width="230" height="64" rx="32" fill="#FFE066" />
          <path d="M100 494l-22-18v36z" fill="#FF9A62" />
        </g>
        <Hub x={146} y={474} size={40} color="#7FD8B6" />
        <Hub x={204} y={474} size={40} color="#7FD8B6" />
        <Hub x={262} y={474} size={40} color="#7FD8B6" />
        <g fill="#FFFFFF" stroke={INK} strokeWidth="2">
          <circle cx="360" cy="470" r="6" />
          <circle cx="374" cy="448" r="4" />
        </g>

        {/* Train */}
        <path d="M450 384H900" stroke={INK} strokeWidth="4" />
        <path d="M470 384v10M500 384v10M530 384v10M560 384v10M590 384v10M620 384v10M650 384v10M680 384v10M710 384v10M740 384v10M770 384v10M800 384v10M830 384v10M860 384v10" stroke={INK} strokeWidth="3" />
        <g stroke={INK} strokeWidth="3.5" strokeLinejoin="round">
          <rect x="470" y="300" width="180" height="70" rx="12" fill="#FFFFFF" />
          <path d="M664 300H800Q870 300 884 370H664Z" fill="#FFFFFF" />
          <rect x="470" y="346" width="180" height="10" fill="#FF9A62" />
          <rect x="664" y="346" width="214" height="10" fill="#FF9A62" />
          <path d="M650 336h14" fill="none" />
          <g fill={INK}>
            <circle cx="500" cy="376" r="9" />
            <circle cx="620" cy="376" r="9" />
            <circle cx="700" cy="376" r="9" />
            <circle cx="830" cy="376" r="9" />
          </g>
        </g>
        <Hub x={490} y={306} size={36} color="#FF9A62" />
        <Hub x={542} y={306} size={36} color="#FF9A62" />
        <Hub x={594} y={306} size={36} color="#FF9A62" />
        <Hub x={684} y={306} size={36} color="#FF9A62" />
        <Hub x={736} y={306} size={36} color="#FF9A62" />

        {/* Plane */}
        <g transform="translate(560 120) scale(1.7)" stroke={INK} strokeWidth="2.4" strokeLinejoin="round" strokeLinecap="round">
          <path d="M30 42L18 10h22l22 28z" fill="#FFB3D1" />
          <path d="M22 46c0-10 18-14 44-14h106c20 0 34 8 38 16-4 8-18 12-38 12H66c-26 0-44-4-44-14z" fill="#FFFFFF" />
          <path d="M100 52l28 28h18l-20-28z" fill="#FFB3D1" />
          <path d="M188 40l12 4-4 3-10-1z" fill={INK} />
        </g>
        <Hub x={684} y={183} size={28} color="#FFB3D1" />
        <Hub x={718} y={183} size={28} color="#FFB3D1" />
        <Hub x={752} y={183} size={28} color="#FFB3D1" />
        <Hub x={786} y={183} size={28} color="#FFB3D1" />
        <Hub x={820} y={183} size={28} color="#FFB3D1" />

        {/* Rocket: the company */}
        <g stroke={INK} strokeWidth="3.5" strokeLinejoin="round">
          <path d="M1060 520V160M1100 520V160M1060 200l40 40M1100 200l-40 40M1060 280l40 40M1100 280l-40 40M1060 360l40 40M1100 360l-40 40M1100 230h40" fill="none" />
          <rect x="1130" y="470" width="180" height="30" rx="4" fill="#6E7179" />
          <path d="M1168 408C1150 440 1196 470 1190 520C1218 486 1222 440 1214 408Z" fill="#FF9A62" />
          <path d="M1182 408C1176 430 1200 452 1196 482C1210 460 1212 432 1206 408Z" fill="#FFE066" />
          <path d="M1140 330L1112 392H1146Z" fill="#FF9A62" />
          <path d="M1240 330L1268 392H1234Z" fill="#FF9A62" />
          <path d="M1190 70C1232 104 1246 160 1246 220V404H1134V220C1134 160 1148 104 1190 70Z" fill="#FFFFFF" />
          <path d="M1162 108C1170 92 1180 80 1190 70C1200 80 1210 92 1218 108Z" fill="#B9AEFF" />
          <rect x="1134" y="372" width="112" height="32" fill={INK} />
          <path d="M1190 404v28" fill="none" />
        </g>
        <Hub x={1150} y={190} size={80} color="#FFE066" />
        <g fill="#FFFFFF" stroke={INK} strokeWidth="3">
          <circle cx="1130" cy="512" r="24" />
          <circle cx="1166" cy="526" r="26" />
          <circle cx="1214" cy="526" r="26" />
          <circle cx="1254" cy="512" r="24" />
          <circle cx="1290" cy="526" r="20" />
        </g>
        <path d="M1278 110v40M1296 140v50M1100 120v36" stroke={INK} strokeWidth="3" strokeLinecap="round" />
      </g>
    </svg>
  )
}
