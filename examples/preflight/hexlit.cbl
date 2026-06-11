       IDENTIFICATION DIVISION.
       PROGRAM-ID. hexlit.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-VAL          PIC 9(5) COMP.
       PROCEDURE DIVISION.
       MAIN.
           COMPUTE WS-VAL = 0xFF + 1
           IF WS-VAL = 256
               DISPLAY 1
           ELSE
               DISPLAY WS-VAL
           END-IF
           COMPUTE WS-VAL = 0x1A
           IF WS-VAL = 26
               DISPLAY 1
           ELSE
               DISPLAY WS-VAL
           END-IF
           STOP RUN.
