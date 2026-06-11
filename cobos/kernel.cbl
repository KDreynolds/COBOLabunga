       IDENTIFICATION DIVISION.
       PROGRAM-ID. COBOLOS.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-INPUT        PIC X(80).
       01 WS-RUNNING      PIC 9(1) COMP VALUE 1.
       01 WS-QEMU-PORT    PIC 9(5) COMP VALUE 0xF4.
       01 WS-QEMU-ZERO    PIC 9(5) COMP VALUE 0.
       PROCEDURE DIVISION.
       KERNEL-MAIN.
           DISPLAY "COBOLOS v0.1"
           DISPLAY "COBOL OPERATING SYSTEM"
           DISPLAY "TYPE HELP FOR HELP"
           DISPLAY " "
           PERFORM UNTIL WS-RUNNING = 0
               PERFORM KERNEL-LOOP
           END-PERFORM
           PERFORM HALT-SYSTEM
           STOP RUN.
       KERNEL-LOOP.
           DISPLAY "> "
           ACCEPT WS-INPUT
           PERFORM HANDLE-INPUT.
       HANDLE-INPUT.
           EVALUATE WS-INPUT
               WHEN "HELLO"
                   DISPLAY "HELLO HUMAN"
               WHEN "HELP"
                   DISPLAY "grug"
               WHEN "QUIT"
                   MOVE 0 TO WS-RUNNING
               WHEN "VERSION"
                   DISPLAY "COBOLOS V0.1"
                   DISPLAY "BUILT WITH COBOLABUNGA"
                   DISPLAY "A COBOL COMPILER WRITTEN IN GO"
                   DISPLAY "TARGETING LLVM IR"
               WHEN OTHER
                   DISPLAY "?"
           END-EVALUATE.
       HALT-SYSTEM.
           PORT-OUT(WS-QEMU-PORT, WS-QEMU-ZERO).
