       IDENTIFICATION DIVISION.
       PROGRAM-ID. compute-test.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A          PIC 9(5).
       01 WS-B          PIC 9(5).
       01 WS-RESULT     PIC 9(5).
       01 WS-GREET      PIC X(30) VALUE "Hello from COBOLabunga!".
       PROCEDURE DIVISION.
       MAIN.
           MOVE 10 TO WS-A
           MOVE 20 TO WS-B
           COMPUTE WS-RESULT = WS-A + WS-B
           DISPLAY WS-GREET
           DISPLAY "A + B = " WS-RESULT
           STOP RUN.
