var gulp = require('gulp');

var coffee = require('gulp-coffee');
var pug = require('gulp-pug');
var gzip = require('gulp-zopfli');
var cleanCSS = require('gulp-clean-css');

var paths = {
    coffee: 'client/**/*.coffee',
    pages:  'templates/**/*.pug'
};

gulp.task('coffee', function() {
    return gulp.src(paths.coffee)
        .pipe(coffee())
        .pipe(gulp.dest('client'));
});

gulp.task('pages', function() {
    return gulp.src(paths.pages)
        .pipe(pug())
        .pipe(gulp.dest('templates'));
});

gulp.task('css', function() {
    return gulp.src('res/css/main.css')
        .pipe(cleanCSS())
        .pipe(gulp.dest('build/css'))
});

gulp.task('copy_fonts', function() {
    return gulp.src('res/fonts/glyphicons-halflings-regular.*')
        .pipe(gulp.dest('serve/fonts'));
});

gulp.task('compress', [
    'coffee',
    'css',
], function() {
    return gulp.src([
            'build/**/*',
            'client/*.js',
            'res/**/*.min.js',
            'res/**/ng-google-chart.js',
            'res/**/ngClip.js',
            'res/**/bootstrap.min.css',
            'res/favicon.ico'
        ])
        .pipe(gzip({ append: false }))
        .pipe(gulp.dest('serve'));
});

gulp.task('watch', [ 'build' ], function() {
    gulp.watch(paths.coffee, [ 'coffee' ]);
    gulp.watch(paths.pages, [ 'pages' ]);
});

gulp.task('build', [
    'pages', 
    'compress',
    'copy_fonts'
]);

gulp.task('default', [ 'build' ]);
