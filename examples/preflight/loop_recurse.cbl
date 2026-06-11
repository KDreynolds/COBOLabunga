       IDENTIFICATION DIVISION.
       PROGRAM-ID. loop-recurse.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-DEPTH        PIC 9(5) COMP VALUE 0.
       PROCEDURE DIVISION.
       MAIN.
           PERFORM LOOP-BODY.
           STOP RUN.
       LOOP-BODY.
           COMPUTE WS-DEPTH = WS-DEPTH + 1
           IF WS-DEPTH < 1000
               PERFORM LOOP-BODY
           END-IF
           DISPLAY WS-DEPTH.
