       IDENTIFICATION DIVISION.
       PROGRAM-ID. evalstr.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-INPUT        PIC X(10).
       PROCEDURE DIVISION.
       MAIN.
           MOVE "CAT" TO WS-INPUT
           EVALUATE WS-INPUT
               WHEN "DOG"
                   DISPLAY 0
               WHEN "CAT"
                   DISPLAY 1
               WHEN OTHER
                   DISPLAY 0
           END-EVALUATE
           STOP RUN.
