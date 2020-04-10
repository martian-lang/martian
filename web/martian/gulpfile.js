var gulp = require('gulp');

var pug = require('gulp-pug');
var gzip = require('gulp-zopfli-green');
var cleanCSS = require('gulp-clean-css');
var concat = require('gulp-concat');

var paths = {
    pages:  'templates/**/*.pug'
};

gulp.task('merge-scripts', function() {
    return gulp.src([
            'node_modules/d3/build/d3.min.js',
            'node_modules/dagre-d3/dist/dagre-d3.min.js',
            'node_modules/angular/angular.min.js',
            'res/js/ui-bootstrap-tpls-0.10.0.min.js',
            'node_modules/lodash/lodash.min.js',
            'res/js/ng-google-chart.js',
            'client/graph.js'])
        .pipe(concat('graph.js'))
        .pipe(gulp.dest('build'));
});

gulp.task('pages', function() {
    return gulp.src(paths.pages)
        .pipe(pug())
        .pipe(gulp.dest('templates'));
});

gulp.task('css', function() {
    return gulp.src('res/css/main.css')
        .pipe(cleanCSS())
        .pipe(gulp.dest('build/css'));
});

gulp.task('run_gzip', function() {
    return gulp.src([
            'build/**/*',
            'res/favicon.ico'
        ])
        .pipe(gzip({ append: false }))
        .pipe(gulp.dest('serve'));
});

gulp.task('compress', gulp.series(gulp.parallel(
    'merge-scripts',
    'css',
), 'run_gzip'));

gulp.task('build', gulp.parallel(
    'pages', 
    'compress',
));

gulp.task('default', gulp.series('build'));
