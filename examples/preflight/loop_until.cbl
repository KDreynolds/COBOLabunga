       IDENTIFICATION DIVISION.
       PROGRAM-ID. loop-until.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-COUNT        PIC 9(5) COMP VALUE 0.
       01 WS-DONE         PIC 9(1) COMP VALUE 0.
       PROCEDURE DIVISION.
       MAIN.
           PERFORM UNTIL WS-DONE = 1
               COMPUTE WS-COUNT = WS-COUNT + 1
               IF WS-COUNT = 5
                   MOVE 1 TO WS-DONE
               END-IF
           END-PERFORM
           DISPLAY WS-COUNT
           STOP RUN.
