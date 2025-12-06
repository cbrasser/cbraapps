I want more features:

## Grade mode ✓

I want to represent student grade in the app as well. this will work as follows:

- a course can have a test. a test has its own data model: ✓
a test has:
- 1 to many questions, each having 1 to many points earnable ✓
- a title and a topic ✓
- each student of the course scores points on each task of the test ✓
- each student gets assigned a grade for this test according to the total points. the formula is (points / max__points * 5 + 1, rounded to quarters). ✓
- a test is added through invoking a command like "cbrateach add-test --points a_csv_file --name test-name --topic test-topic" or something like that. ✓
- a test is in "review mode" by default and can be viewed in a separate view. Here, the teachers sees a table with the student names and points per task, total points, average grades, and so on. ✓
- the user can "confirm" a test, marking the grades as finished and moving the test out of review mode. ✓
- in review mode, the teacher can define a number of "gifted" points. these will be subtracted from the total points in the grade calculation. ✓ (subtracted from max_points for easier calculation)
- inline table editing supported in review mode ✓
- uses Bubbles table component for professional display ✓

there is an export function in the course view that will export the a csv file with columns: name, surname, grade. for the course. ✓

## UI Enhancements ✓

### Test Review View ✓
- Missing students below test table with clear list ✓
- Editing cells highlighted in orange background (#FFA500) ✓
- Vertical bar chart for grade distribution (Verteilung) ✓
- Weight displayed in statistics line ✓

### Course Management ✓
- Course details editable via 'd' key in classbook view ✓
- Topic tags displayed as styled badges in course list ✓
- Direct test access with 't' shortcut from course list ✓

### Student View ✓
- Test grades displayed in classbook for selected student ✓
- Weighted average grade calculated and displayed ✓
- Status icons: 📝 for review, ✓ for confirmed tests ✓

### Test Weighting ✓
- Tests can have weights via --test-weight flag ✓
- Weighted average calculation for final grades ✓
- Default weight of 1.0 if not specified ✓
