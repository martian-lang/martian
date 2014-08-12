var gulp = require('gulp');

var coffee = require('gulp-coffee');
var jade = require('gulp-jade');

var paths = {
    coffee: 'client/**/*.coffee',
    pages:  'templates/**/*.jade'
};

gulp.task('coffee', function() {
    return gulp.src(paths.coffee)
        .pipe(coffee())
        .pipe(gulp.dest('client'));
});

gulp.task('pages', function() {
    return gulp.src(paths.pages)
        .pipe(jade({pretty:true}))
        .pipe(gulp.dest('templates'));
});

gulp.task('watch', [ 'build' ], function() {
    gulp.watch(paths.coffee, [ 'coffee' ]);
    gulp.watch(paths.pages, [ 'pages' ]);
});

gulp.task('build', [
    'coffee', 
    'pages', 
]);

gulp.task('default', [ 'build' ]);
