% Generated by roxygen2: do not edit by hand
% Please edit documentation in R/Arvados.R
\name{projects.update}
\alias{projects.update}
\title{project.update}
\usage{
arv$projects.update(group, uuid)
}
\arguments{
\item{group}{Group object.}

\item{uuid}{The UUID of the Group in question.}
}
\value{
Group object.
}
\description{
projects.update wrapps groups.update method by setting group_class attribute to "project".
}
