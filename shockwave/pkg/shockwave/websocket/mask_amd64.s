// +build amd64,!noasm

#include "textflag.h"

// func maskBytesAVX2(data []byte, mask [4]byte)
TEXT ·maskBytesAVX2(SB), NOSPLIT, $0-36
    MOVQ    data_base+0(FP), DI  // DI = data pointer
    MOVQ    data_len+8(FP), CX   // CX = data length
    MOVL    mask+24(FP), AX      // AX = mask key (4 bytes)

    // Check if we have AVX2 support (done at init time)
    // For now, assume it's available if this function is called

    // Broadcast mask to 32 bytes for AVX2
    // Create mask pattern: [m0 m1 m2 m3 m0 m1 m2 m3 ...]
    MOVL    AX, BX
    SHLL    $8, BX
    ORL     BX, AX
    SHLL    $8, BX
    ORL     BX, AX
    SHLL    $8, BX
    ORL     BX, AX

    // Load mask into XMM register
    MOVD    AX, X0
    VPBROADCASTD X0, Y0  // Broadcast to YMM register (32 bytes)

    // Process 32 bytes at a time with AVX2
    CMPQ    CX, $32
    JB      tail

avx2_loop:
    VMOVDQU (DI), Y1     // Load 32 bytes
    VPXOR   Y0, Y1, Y1   // XOR with mask
    VMOVDQU Y1, (DI)     // Store 32 bytes
    ADDQ    $32, DI
    SUBQ    $32, CX
    CMPQ    CX, $32
    JAE     avx2_loop

tail:
    // Process remaining bytes (< 32)
    VZEROUPPER  // Clear upper 128 bits of YMM registers

    // Fall back to scalar for remaining bytes
    CMPQ    CX, $8
    JB      byte_loop

qword_loop:
    MOVQ    (DI), BX
    XORQ    AX, BX
    MOVQ    BX, (DI)
    ADDQ    $8, DI
    SUBQ    $8, CX
    CMPQ    CX, $8
    JAE     qword_loop

byte_loop:
    CMPQ    CX, $0
    JE      done

    MOVB    (DI), BX
    XORB    AX, BX
    MOVB    BX, (DI)

    // Rotate mask for next byte
    RORL    $8, AX

    INCQ    DI
    DECQ    CX
    JMP     byte_loop

done:
    RET
